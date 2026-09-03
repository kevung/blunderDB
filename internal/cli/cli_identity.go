package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/issuance"
)

// runOpen turns a protected copy back into an ordinary database. It is the one time a
// password is ever asked for: from then on the recipient works with a normal file.
func (cli *CLI) runOpen(args []string) error {
	openCmd := flag.NewFlagSet("open", flag.ContinueOnError)

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
		fmt.Println("  blunderdb open --db cours.dbx --password secret")
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

// runIdentity shows and moves the Issuer identity — the durable key every watermark is
// signed with. It belongs to a person, not to a database, which is why it lives in the
// config directory and travels as a single file.
func (cli *CLI) runIdentity(args []string) error {
	idCmd := flag.NewFlagSet("identity", flag.ContinueOnError)

	name := idCmd.String("name", "", "Change the display name carried by future watermarks")
	exportPath := idCmd.String("export", "", "Write your identity to a file, to carry to another machine")
	importPath := idCmd.String("import", "", "Install an identity file on this machine")
	passphrase := idCmd.String("passphrase", "", "Passphrase for the exported/imported file (optional)")
	format := idCmd.String("format", "text", "Output format: text or json")

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

	formatLower := strings.ToLower(*format)
	if formatLower != "text" && formatLower != "json" {
		return fmt.Errorf("unknown format: %s (must be 'text' or 'json')", *format)
	}
	text := formatLower != "json"

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
		if !text {
			return printJSON(struct {
				Name        string `json:"name"`
				Fingerprint string `json:"fingerprint"`
			}{Name: id.Name, Fingerprint: id.Fingerprint()})
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
		if text {
			fmt.Printf("Identity written to %s\n", *exportPath)
			if *passphrase == "" {
				fmt.Println("It is NOT protected: anyone holding this file can sign in your name.")
			}
			fmt.Println()
		}
	}

	if !text {
		return printJSON(struct {
			Name        string `json:"name"`
			Fingerprint string `json:"fingerprint"`
			StoredIn    string `json:"stored_in"`
			ExportedTo  string `json:"exported_to,omitempty"`
		}{
			Name:        identity.Name,
			Fingerprint: identity.Fingerprint(),
			StoredIn:    issuance.IdentityPath(issuance.ConfigDir()),
			ExportedTo:  *exportPath,
		})
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
