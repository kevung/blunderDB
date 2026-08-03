package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// This file is the database glue for pkg/blunderdb/issuance: reading and writing the four
// issuance documents that live in `metadata`, producing Issued copies, recording Holders,
// and inheriting Lineage on import. All the reasoning lives in the issuance package doc and
// in docs/adr/0007-issued-copies-are-attributable-never-unshareable.md; what follows is
// plumbing.

// readMeta returns a metadata value, treating an absent row as empty. Issuance documents are
// absent from most databases — never issued, never imported an Issued copy — so absence is
// the normal case and not an error.
func (d *Database) readMeta(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// writeMeta stores a metadata value, deleting the row when the value is empty so a database
// that carries no issuance never grows the rows for it.
func (d *Database) writeMeta(key, value string) error {
	if value == "" {
		_, err := d.db.Exec(`DELETE FROM metadata WHERE key = ?`, key)
		return err
	}
	_, err := d.db.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, key, value)
	return err
}

// issuanceDocuments is the raw state, read under the caller's lock.
type issuanceDocuments struct {
	watermark issuance.Envelope
	holders   issuance.Registry
	lineage   issuance.Lineage
	register  issuance.IssueRegister
}

func (d *Database) readIssuance() (issuanceDocuments, error) {
	var docs issuanceDocuments
	raw, err := d.readMeta(issuance.KeyWatermark)
	if err != nil {
		return docs, err
	}
	if docs.watermark, err = issuance.DecodeEnvelope(raw); err != nil {
		return docs, err
	}
	if raw, err = d.readMeta(issuance.KeyHolders); err != nil {
		return docs, err
	}
	if docs.holders, err = issuance.DecodeRegistry(raw); err != nil {
		return docs, err
	}
	if raw, err = d.readMeta(issuance.KeyLineage); err != nil {
		return docs, err
	}
	if docs.lineage, err = issuance.DecodeLineage(raw); err != nil {
		return docs, err
	}
	if raw, err = d.readMeta(issuance.KeyIssued); err != nil {
		return docs, err
	}
	docs.register, err = issuance.DecodeRegister(raw)
	return docs, err
}

// GetIssuanceInfo reports how this database was handed out and where its contents came
// from. It never writes: examining a copy must not add the examiner's own machine to the
// evidence.
func (d *Database) GetIssuanceInfo() (IssuanceInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return IssuanceInfo{}, fmt.Errorf("no database is currently open")
	}
	docs, err := d.readIssuance()
	if err != nil {
		return IssuanceInfo{}, err
	}

	// The identity is only needed to answer "did I issue this?"; a machine that has never
	// issued anything simply answers no, and must not have an identity created for it as a
	// side effect of looking at a panel.
	var mine string
	if id, err := issuance.LoadIdentity(issuance.ConfigDir()); err == nil && id != nil {
		mine = id.Fingerprint()
	}

	info := IssuanceInfo{
		IsIssuedCopy:      docs.watermark.IsIssued(),
		ChainIntact:       docs.holders.ChainIntact(docs.watermark.Signature),
		IssuerFingerprint: mine,
	}
	if id, err := issuance.LoadIdentity(issuance.ConfigDir()); err == nil && id != nil {
		info.IssuerName = id.Name
	}
	if info.IsIssuedCopy {
		wm := watermarkInfo(docs.watermark, mine)
		info.Watermark = &wm
	}
	for _, h := range docs.holders.Holders {
		info.Holders = append(info.Holders, HolderInfo{
			Fingerprint: h.Fingerprint, FirstSeen: h.FirstSeen, LastSeen: h.LastSeen, Openings: h.Openings,
		})
	}
	for _, e := range docs.lineage {
		info.Lineage = append(info.Lineage, watermarkInfo(e, mine))
	}
	for _, r := range docs.register.Records {
		info.Issued = append(info.Issued, IssueRecordInfo{
			Distribution: r.Distribution, Recipient: r.Recipient, Number: r.Number, Total: r.Total,
			IssuedAt: r.IssuedAt, FileName: r.FileName, Contents: r.Contents,
			Password: r.Password, Signature: r.Signature,
		})
	}
	return info, nil
}

func watermarkInfo(e issuance.Envelope, myFingerprint string) WatermarkInfo {
	w, err := e.Open()
	info := WatermarkInfo{
		IssuerFingerprint: e.Fingerprint(),
		SignatureValid:    e.Verify(),
	}
	if err != nil {
		return info
	}
	info.Distribution = w.Distribution
	info.IssuerName = w.IssuerName
	info.Recipient = w.Recipient
	info.Number = w.Number
	info.Total = w.Total
	info.IssuedAt = w.IssuedAt
	info.Nominative = w.Nominative()
	info.IssuedByYou = myFingerprint != "" && e.VerifiedBy(myFingerprint)
	return info
}

// RecordHolder notes that this machine has opened the database as a working copy.
//
// It is deliberately **not** called from OpenDatabase, which the CLI and the daemon also
// run: a Holder registry that grew every time a file was inspected would write the
// examiner's own machine into the evidence, and would turn every scripted `blunderdb`
// invocation into a phantom holder. Only the desktop GUI calls this, once per open.
//
// A database that is not an Issued copy is left completely alone — no rows, no trace.
func (d *Database) RecordHolder() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	if d.readOnly {
		// Opened alongside another instance; the writer owns the registry.
		return nil
	}
	docs, err := d.readIssuance()
	if err != nil {
		return err
	}
	if !docs.watermark.IsIssued() {
		return nil
	}
	w, err := docs.watermark.Open()
	if err != nil {
		// An unreadable watermark is not a reason to refuse to open a database.
		slog.Warn("unreadable watermark, holder not recorded", "err", err)
		return nil
	}
	docs.holders.Record(docs.watermark.Signature, issuance.MachineFingerprint(w.Salt), time.Now())
	encoded, err := issuance.EncodeRegistry(docs.holders)
	if err != nil {
		return err
	}
	return d.writeMeta(issuance.KeyHolders, encoded)
}

// IssuerIdentity returns this machine's Issuer identity, creating it silently if this is the
// first emission. Nothing in the UI asks the user to configure it.
func IssuerIdentity() (*issuance.Identity, error) {
	return issuance.LoadOrCreateIdentity(issuance.ConfigDir(), issuance.DefaultIssuerName())
}

// IssueCopies produces the Issued copies of one Distribution and records them in this
// database's Issue register.
//
// One file per recipient in the nominative regime, a single file when no recipient is
// named. Copies of one Distribution share a salt — issuing a second batch for the same
// course reuses the salt and continues the numbering — so their Holder fingerprints stay
// comparable with each other.
func (d *Database) IssueCopies(opts ExportOptions, iss IssuanceOptions) ([]IssuedCopy, error) {
	if strings.TrimSpace(iss.Distribution) == "" {
		return nil, fmt.Errorf("a distribution name is required to issue copies")
	}
	identity, err := IssuerIdentity()
	if err != nil {
		return nil, err
	}

	// Read what the source carries before producing anything: its own Watermark (if it is
	// itself an Issued copy) and its Lineage both travel into every copy, so a course a
	// colleague passed on keeps its trace.
	d.mu.RLock()
	if d.db == nil {
		d.mu.RUnlock()
		return nil, fmt.Errorf("no database is currently open")
	}
	// Refuse up front rather than half-way: a read-only source (another instance holds the
	// write lock) can produce perfectly valid copies but cannot record them in its issue
	// register. Copies that exist and are in no register are worse than no copies — the
	// issuer would have handed out files they can no longer look up.
	readOnly := d.readOnly
	docs, err := d.readIssuance()
	d.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	if readOnly {
		return nil, fmt.Errorf("this database is open read-only, so issued copies could not be recorded in its issue register; close the other blunderDB instance and try again")
	}

	inherited := issuance.Lineage(nil).Inherit(docs.watermark, docs.lineage)
	lineageDoc, err := issuance.EncodeLineage(inherited)
	if err != nil {
		return nil, err
	}

	recipients := cleanRecipients(iss.Recipients)
	collective := len(recipients) == 0
	if collective {
		recipients = []string{""}
	}

	salt := docs.register.SaltFor(iss.Distribution)
	first := docs.register.NextNumber(iss.Distribution)
	total := len(recipients)
	if !collective {
		total = first - 1 + len(recipients)
	}

	var produced []IssuedCopy
	var records []issuance.IssueRecord

	for i, recipient := range recipients {
		number := first + i
		env, err := issuance.Seal(identity, issuance.Watermark{
			Distribution: iss.Distribution,
			IssuerName:   identity.Name,
			Recipient:    recipient,
			Number:       number,
			Total:        total,
			Salt:         salt,
		})
		if err != nil {
			return produced, err
		}
		if salt == "" {
			// The first copy of a Distribution mints the salt; every later one reuses it.
			w, err := env.Open()
			if err != nil {
				return produced, err
			}
			salt = w.Salt
		}
		encodedWatermark, err := issuance.EncodeEnvelope(env)
		if err != nil {
			return produced, err
		}

		path, err := copyPath(opts.ExportPath, iss, recipient, collective)
		if err != nil {
			return produced, err
		}

		copyOpts := opts
		copyOpts.ExportPath = path
		copyOpts.WatermarkDocument = encodedWatermark
		copyOpts.LineageDocument = lineageDoc
		if iss.Password != "" {
			// Export to a plain file first, then wrap it; the intermediate is removed
			// whether or not wrapping succeeds.
			copyOpts.ExportPath = path + ".plain"
		}
		if err := d.ExportDatabase(copyOpts); err != nil {
			return produced, err
		}
		if iss.Password != "" {
			wrapErr := issuance.WrapContainer(copyOpts.ExportPath, path, env, iss.Password)
			if rmErr := os.Remove(copyOpts.ExportPath); rmErr != nil {
				slog.Warn("removing the intermediate export", "path", copyOpts.ExportPath, "err", rmErr)
			}
			if wrapErr != nil {
				return produced, wrapErr
			}
		}

		produced = append(produced, IssuedCopy{
			Path: path, Recipient: recipient, Number: number, Total: total,
			Encrypted: iss.Password != "", Signature: env.Signature,
		})
		records = append(records, issuance.IssueRecord{
			Distribution: iss.Distribution, Recipient: recipient, Number: number, Total: total,
			CopyID: copyIDOf(env), Signature: env.Signature, Salt: salt,
			FileName: filepath.Base(path), Contents: iss.Contents, Password: iss.Password,
		})
	}

	if err := d.appendIssueRecords(records); err != nil {
		// The files exist and are valid; failing to note them locally is worth reporting
		// but must not be mistaken for "nothing was issued".
		return produced, fmt.Errorf("copies were issued but the issue register could not be updated: %w", err)
	}
	return produced, nil
}

func copyIDOf(e issuance.Envelope) string {
	if w, err := e.Open(); err == nil {
		return w.CopyID
	}
	return ""
}

func (d *Database) appendIssueRecords(records []issuance.IssueRecord) error {
	if len(records) == 0 {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	raw, err := d.readMeta(issuance.KeyIssued)
	if err != nil {
		return err
	}
	reg, err := issuance.DecodeRegister(raw)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, rec := range records {
		reg.Add(rec, now)
	}
	encoded, err := issuance.EncodeRegister(reg)
	if err != nil {
		return err
	}
	return d.writeMeta(issuance.KeyIssued, encoded)
}

func cleanRecipients(in []string) []string {
	var out []string
	for _, r := range in {
		if r = strings.TrimSpace(r); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// copyPath decides where one Issued copy is written: the caller's chosen file for a single
// copy, or a generated name inside the output directory for a batch.
func copyPath(exportPath string, iss IssuanceOptions, recipient string, collective bool) (string, error) {
	if iss.OutputDir == "" {
		if exportPath == "" {
			return "", fmt.Errorf("no destination for the issued copy")
		}
		if iss.Password != "" && !strings.HasSuffix(exportPath, issuance.ContainerExtension) {
			return strings.TrimSuffix(exportPath, ".db") + issuance.ContainerExtension, nil
		}
		return exportPath, nil
	}
	if err := os.MkdirAll(iss.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create the destination folder: %w", err)
	}
	name := issuance.CopyFileName(iss.Distribution, recipient, iss.Password != "")
	if collective {
		name = issuance.CopyFileName(iss.Distribution, "", iss.Password != "")
	}
	return filepath.Join(iss.OutputDir, name), nil
}

// inheritLineageTx records that this database now contains material from an Issued copy. It
// runs inside the import's own transaction so the trace lands atomically with the positions
// it describes — a cancelled import must not leave a lineage entry for content it rolled
// back.
//
// Importing a course into one's own database is a normal, supported gesture — and the
// easiest way to strip a Watermark, since positions travel by Zobrist hash while `metadata`
// does not. Carrying the source's Watermark forward keeps the trace attached to the result,
// so the default path through the product does not launder it. It is written silently: the
// design trades the deterrent effect for discretion, and documents the mechanism in the
// manual instead.
//
// Failing to inherit is never a reason to fail an import: a source with no readable
// issuance is simply a source that was never issued, which is the overwhelmingly common
// case.
func (d *Database) inheritLineageTx(tx *sql.Tx, sourcePath string) {
	sourceWatermark, sourceLineage, err := readIssuanceFrom(sourcePath)
	if err != nil {
		slog.Debug("no issuance documents in imported database", "path", sourcePath, "err", err)
		return
	}
	if !sourceWatermark.IsIssued() && len(sourceLineage) == 0 {
		return
	}

	var raw string
	if err := tx.QueryRow(`SELECT value FROM metadata WHERE key = ?`, issuance.KeyLineage).Scan(&raw); err != nil && err != sql.ErrNoRows {
		slog.Warn("reading lineage before import", "err", err)
		return
	}
	current, err := issuance.DecodeLineage(raw)
	if err != nil {
		slog.Warn("unreadable lineage, starting a new one", "err", err)
		current = nil
	}
	encoded, err := issuance.EncodeLineage(current.Inherit(sourceWatermark, sourceLineage))
	if err != nil {
		slog.Warn("encoding lineage after import", "err", err)
		return
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`, issuance.KeyLineage, encoded); err != nil {
		slog.Warn("writing lineage after import", "err", err)
	}
}

// readIssuanceFrom peeks at another database file's issuance documents without opening it
// through the Database wrapper — no migration, no locking, no side effects on the file.
func readIssuanceFrom(path string) (issuance.Envelope, issuance.Lineage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return issuance.Envelope{}, nil, err
	}
	defer db.Close()

	read := func(key string) string {
		var value string
		if err := db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value); err != nil {
			return ""
		}
		return value
	}
	env, err := issuance.DecodeEnvelope(read(issuance.KeyWatermark))
	if err != nil {
		return issuance.Envelope{}, nil, err
	}
	lin, err := issuance.DecodeLineage(read(issuance.KeyLineage))
	if err != nil {
		return env, nil, err
	}
	return env, lin, nil
}

// IsProtectedCopy reports whether a path is a password-protected copy rather than an
// ordinary database, so the caller can ask for the password instead of failing to open a
// file that was never meant to be opened directly.
func IsProtectedCopy(path string) bool { return issuance.IsContainer(path) }

// OpenProtectedCopy turns a protected copy into an ordinary database and returns its path.
// This is the one time a password is ever asked for: from then on the recipient works with
// a normal file, with no prompt and no re-encryption.
//
// The result lands beside the container with a `.db` extension. An existing file is never
// overwritten — a second open would otherwise silently discard the work already done in the
// database from the first one.
func OpenProtectedCopy(path, password string) (string, error) {
	target := issuance.DefaultUnwrapPath(path)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	if _, err := issuance.UnwrapContainer(path, target, password); err != nil {
		return "", err
	}
	return target, nil
}

// IsProtectedCopyPath is the bound form of IsProtectedCopy, for the frontend.
func (d *Database) IsProtectedCopyPath(path string) bool { return IsProtectedCopy(path) }

// OpenProtectedCopyPath is the bound form of OpenProtectedCopy, for the frontend. A wrong
// password comes back as a plain message the UI can show as-is.
func (d *Database) OpenProtectedCopyPath(path, password string) (string, error) {
	return OpenProtectedCopy(path, password)
}

// InspectIssuance reports the issuance documents of a file on disk without opening it as a
// working database. This is the forensic path: a copy that comes back must be examinable
// without writing anything into it.
//
// It reads an encrypted copy's cleartext header too — a protected copy found in the wild is
// identifiable without its password, which is the whole reason the header is not encrypted.
func InspectIssuance(path string) (IssuanceInfo, error) {
	var mine string
	if id, err := issuance.LoadIdentity(issuance.ConfigDir()); err == nil && id != nil {
		mine = id.Fingerprint()
	}

	if issuance.IsContainer(path) {
		header, err := issuance.ReadContainerHeader(path)
		if err != nil {
			return IssuanceInfo{}, err
		}
		info := IssuanceInfo{IsIssuedCopy: header.Watermark.IsIssued(), IssuerFingerprint: mine}
		if info.IsIssuedCopy {
			wm := watermarkInfo(header.Watermark, mine)
			info.Watermark = &wm
		}
		return info, nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return IssuanceInfo{}, err
	}
	defer db.Close()

	read := func(key string) string {
		var value string
		if err := db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value); err != nil {
			return ""
		}
		return value
	}
	env, err := issuance.DecodeEnvelope(read(issuance.KeyWatermark))
	if err != nil {
		return IssuanceInfo{}, err
	}
	holders, err := issuance.DecodeRegistry(read(issuance.KeyHolders))
	if err != nil {
		return IssuanceInfo{}, err
	}
	lineage, err := issuance.DecodeLineage(read(issuance.KeyLineage))
	if err != nil {
		return IssuanceInfo{}, err
	}
	// The issue register is read here too: `info` is normally pointed at one's own course
	// database, and the register is the list of copies produced from it. It exists in no
	// other file, since it never travels inside an issued copy.
	register, err := issuance.DecodeRegister(read(issuance.KeyIssued))
	if err != nil {
		return IssuanceInfo{}, err
	}

	info := IssuanceInfo{
		IsIssuedCopy:      env.IsIssued(),
		ChainIntact:       holders.ChainIntact(env.Signature),
		IssuerFingerprint: mine,
	}
	if info.IsIssuedCopy {
		wm := watermarkInfo(env, mine)
		info.Watermark = &wm
	}
	for _, h := range holders.Holders {
		info.Holders = append(info.Holders, HolderInfo{
			Fingerprint: h.Fingerprint, FirstSeen: h.FirstSeen, LastSeen: h.LastSeen, Openings: h.Openings,
		})
	}
	for _, e := range lineage {
		info.Lineage = append(info.Lineage, watermarkInfo(e, mine))
	}
	for _, r := range register.Records {
		info.Issued = append(info.Issued, IssueRecordInfo{
			Distribution: r.Distribution, Recipient: r.Recipient, Number: r.Number, Total: r.Total,
			IssuedAt: r.IssuedAt, FileName: r.FileName, Contents: r.Contents,
			Password: r.Password, Signature: r.Signature,
		})
	}
	return info, nil
}
