package cli

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// runIssue produces Issued copies of a database — the emission path, in the CLI.
//
// Watermarking happens here and only here: editing a database in place and sending the file
// cannot produce an Issued copy, because nothing in that gesture says who it is for.
func (cli *CLI) runIssue(args []string) error {
	issueCmd := flag.NewFlagSet("issue", flag.ExitOnError)

	dbPath := issueCmd.String("db", "", "Path to the database to issue from (required)")
	distribution := issueCmd.String("distribution", "", "Name of the distribution — the lesson or occasion (required)")
	to := issueCmd.String("to", "", "Comma-separated recipients, one copy each (empty = one collective copy)")
	toFile := issueCmd.String("to-file", "", "File with one recipient per line, instead of --to")
	outputDir := issueCmd.String("dir", "", "Folder to write the copies into (required for more than one recipient)")
	outputFile := issueCmd.String("file", "", "Path of the single copy to produce (when there is only one)")
	password := issueCmd.String("password", "", "Protect each copy with a password (protects transport, not the database)")
	contents := issueCmd.String("contents", "", "What this emission contains, noted in your issue register")

	includeAnalysis := issueCmd.Bool("analysis", true, "Include analysis")
	includeComments := issueCmd.Bool("comments", true, "Include comments")
	includeFilterLibrary := issueCmd.Bool("filters", true, "Include the filter library")
	includePlayedMoves := issueCmd.Bool("played-moves", true, "Include played moves in analysis")
	includeMatches := issueCmd.Bool("matches", true, "Include matches")

	issueCmd.Usage = func() {
		fmt.Println("Usage: blunderdb issue [options]")
		fmt.Println()
		fmt.Println("Produce watermarked copies of a database to hand out.")
		fmt.Println()
		fmt.Println("Each copy carries a watermark signed with your issuer identity, saying which")
		fmt.Println("distribution it belongs to and — when you name recipients — who it was made for.")
		fmt.Println("A watermark is tamper-evident and unforgeable; it is NOT unremovable, and it")
		fmt.Println("prevents nothing. It lets a copy that leaks be traced back to one emission.")
		fmt.Println()
		fmt.Println("Options:")
		issueCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  # One copy per student, each attributable")
		fmt.Println("  blunderdb issue --db cours.db --distribution \"Cours du 12 mars\" \\")
		fmt.Println("      --to \"Kévin Unger,Marie Durand\" --dir ./exemplaires")
		fmt.Println()
		fmt.Println("  # A recipient list from a file")
		fmt.Println("  blunderdb issue --db cours.db --distribution \"Cours du 12 mars\" \\")
		fmt.Println("      --to-file eleves.txt --dir ./exemplaires")
		fmt.Println()
		fmt.Println("  # A single collective copy for a whole group")
		fmt.Println("  blunderdb issue --db cours.db --distribution \"Promotion 2026\" --file cours.db")
		fmt.Println()
		fmt.Println("  # Protected in transit (the password is kept in your issue register)")
		fmt.Println("  blunderdb issue --db cours.db --distribution \"Cours du 12 mars\" \\")
		fmt.Println("      --to-file eleves.txt --dir ./exemplaires --password secret")
	}

	if err := issueCmd.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		issueCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if strings.TrimSpace(*distribution) == "" {
		issueCmd.Usage()
		return fmt.Errorf("missing required flag: --distribution")
	}

	recipients, err := gatherRecipients(*to, *toFile)
	if err != nil {
		return err
	}
	if len(recipients) > 1 && *outputDir == "" {
		return fmt.Errorf("--dir is required when issuing to more than one recipient")
	}
	if len(recipients) <= 1 && *outputDir == "" && *outputFile == "" {
		return fmt.Errorf("give either --file (one copy) or --dir (a folder of copies)")
	}

	if err := cli.initDatabase(*dbPath); err != nil {
		return err
	}

	positions, err := cli.db.LoadAllPositions()
	if err != nil {
		return fmt.Errorf("failed to load positions: %v", err)
	}
	metadata, err := cli.db.LoadMetadata()
	if err != nil {
		metadata = make(map[string]string)
	}

	copies, err := cli.db.IssueCopies(ExportOptions{
		ExportPath:           *outputFile,
		Positions:            positions,
		Metadata:             metadata,
		IncludeAnalysis:      *includeAnalysis,
		IncludeComments:      *includeComments,
		IncludeFilterLibrary: *includeFilterLibrary,
		IncludePlayedMoves:   *includePlayedMoves,
		IncludeMatches:       *includeMatches,
	}, database.IssuanceOptions{
		Distribution: strings.TrimSpace(*distribution),
		Recipients:   recipients,
		OutputDir:    *outputDir,
		Password:     *password,
		Contents:     *contents,
	})
	if err != nil {
		return fmt.Errorf("failed to issue copies: %v", err)
	}

	identity, idErr := database.IssuerIdentity()
	if idErr == nil {
		fmt.Printf("Issued as %s (%s)\n", identity.Name, identity.Fingerprint())
	}
	fmt.Printf("Distribution: %s\n\n", strings.TrimSpace(*distribution))
	for _, c := range copies {
		who := c.Recipient
		if who == "" {
			who = "(collective)"
		}
		fmt.Printf("  %3d/%-3d  %-24s  %s\n", c.Number, c.Total, who, c.Path)
	}
	noun := "copies"
	if len(copies) == 1 {
		noun = "copy"
	}
	fmt.Printf("\n%d %s written. They are recorded in the issue register of %s,\n", len(copies), noun, *dbPath)
	fmt.Println("which never travels inside a copy.")
	if *password != "" {
		fmt.Println("\nThe password protects the copies in transit only — the recipient holds it.")
		fmt.Println("Recipients open them with: blunderdb open --db <copy> --password <password>")
	}
	return nil
}

// runOpen turns a protected copy back into an ordinary database. It is the one time a
// password is ever asked for: from then on the recipient works with a normal file.
func (cli *CLI) runOpen(args []string) error {
	openCmd := flag.NewFlagSet("open", flag.ExitOnError)

	path := openCmd.String("db", "", "Path to the protected copy (required)")
	password := openCmd.String("password", "", "The password you were given (required)")
	outFile := openCmd.String("file", "", "Where to write the opened database (default: alongside, with .db)")

	openCmd.Usage = func() {
		fmt.Println("Usage: blunderdb open [options]")
		fmt.Println()
		fmt.Println("Open a password-protected copy into an ordinary database file.")
		fmt.Println()
		fmt.Println("You are asked for the password once. The result is a normal blunderDB")
		fmt.Println("database you work with as usual.")
		fmt.Println()
		fmt.Println("Options:")
		openCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  blunderdb open --db cours.bdbx --password secret")
	}

	if err := openCmd.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		openCmd.Usage()
		return fmt.Errorf("missing required flag: --db")
	}
	if !issuance.IsContainer(*path) {
		return fmt.Errorf("%s is not a protected copy — open it as an ordinary database", *path)
	}
	if *password == "" {
		openCmd.Usage()
		return fmt.Errorf("missing required flag: --password")
	}

	target := *outFile
	if target == "" {
		target = issuance.DefaultUnwrapPath(*path)
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("%s already exists — pass --file to choose another name", target)
	}
	if _, err := issuance.UnwrapContainer(*path, target, *password); err != nil {
		return err
	}
	fmt.Printf("Opened into %s\n", target)
	return nil
}

func gatherRecipients(inline, fromFile string) ([]string, error) {
	var names []string
	for _, n := range strings.Split(inline, ",") {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	if fromFile == "" {
		return names, nil
	}
	f, err := os.Open(fromFile)
	if err != nil {
		return nil, fmt.Errorf("cannot read the recipient list: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if n := strings.TrimSpace(scanner.Text()); n != "" && !strings.HasPrefix(n, "#") {
			names = append(names, n)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot read the recipient list: %v", err)
	}
	return names, nil
}

// runIdentity shows and moves the Issuer identity — the durable key every watermark is
// signed with. It belongs to a person, not to a database, which is why it lives in the
// config directory and travels as a single file.
func (cli *CLI) runIdentity(args []string) error {
	idCmd := flag.NewFlagSet("identity", flag.ExitOnError)

	name := idCmd.String("name", "", "Change the display name carried by future watermarks")
	exportPath := idCmd.String("export", "", "Write your identity to a file, to carry to another machine")
	importPath := idCmd.String("import", "", "Install an identity file on this machine")
	passphrase := idCmd.String("passphrase", "", "Passphrase for the exported/imported file (optional)")

	idCmd.Usage = func() {
		fmt.Println("Usage: blunderdb identity [options]")
		fmt.Println()
		fmt.Println("Show or move your issuer identity.")
		fmt.Println()
		fmt.Println("The identity is created by itself the first time you issue copies; you never")
		fmt.Println("have to set it up. Copies you have already issued keep verifying whatever you")
		fmt.Println("do here — the name is only a label, and the key is what signs.")
		fmt.Println()
		fmt.Println("The exported file lets anyone holding it sign in your name. Do not share it.")
		fmt.Println()
		fmt.Println("Options:")
		idCmd.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  blunderdb identity")
		fmt.Println("  blunderdb identity --name \"Jean Dupont\"")
		fmt.Println("  blunderdb identity --export jean.bdbid --passphrase secret")
		fmt.Println("  blunderdb identity --import jean.bdbid --passphrase secret")
	}

	if err := idCmd.Parse(args); err != nil {
		return err
	}

	if *importPath != "" {
		needs, err := issuance.IdentityFileNeedsPassphrase(*importPath)
		if err != nil {
			return err
		}
		if needs && *passphrase == "" {
			return fmt.Errorf("this identity file is protected: pass --passphrase")
		}
		id, err := issuance.ImportIdentity(issuance.ConfigDir(), *importPath, *passphrase)
		if err != nil {
			return err
		}
		fmt.Printf("Identity installed: %s (%s)\n", id.Name, id.Fingerprint())
		return nil
	}

	identity, err := database.IssuerIdentity()
	if err != nil {
		return err
	}
	if *name != "" {
		if err := identity.Rename(issuance.ConfigDir(), *name); err != nil {
			return err
		}
	}
	if *exportPath != "" {
		if err := identity.ExportIdentity(*exportPath, *passphrase); err != nil {
			return err
		}
		fmt.Printf("Identity written to %s\n", *exportPath)
		if *passphrase == "" {
			fmt.Println("It is NOT protected: anyone holding this file can sign in your name.")
		}
		fmt.Println()
	}

	fmt.Println("Issuer identity")
	fmt.Printf("  Name:        %s\n", identity.Name)
	fmt.Printf("  Fingerprint: %s\n", identity.Fingerprint())
	fmt.Printf("  Stored in:   %s\n", issuance.IdentityPath(issuance.ConfigDir()))
	fmt.Println()
	fmt.Println("Give the fingerprint to your recipients if you want them to be able to check")
	fmt.Println("that a copy really came from you.")
	return nil
}
