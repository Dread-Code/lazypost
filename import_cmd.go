package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Dread-Code/lazypost/internal/app"
	"github.com/Dread-Code/lazypost/internal/importer"
)

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	target := fs.String("dir", "", "target collection directory")
	format := fs.String("format", "", "source format: postman or insomnia (auto-detected by default)")
	dryRun := fs.Bool("dry-run", false, "parse and report without writing")
	force := fs.Bool("force", false, "replace an existing target directory")
	strict := fs.Bool("strict", false, "fail when any warning is produced")
	var environmentFiles []string
	fs.Func("env", "Postman or Insomnia environment file; repeatable", func(value string) error {
		environmentFiles = append(environmentFiles, value)
		return nil
	})

	// Go's flag package stops parsing at the first positional argument. The
	// documented syntax puts the source first, so remove it before parsing
	// flags while still accepting the conventional flags-first form.
	source := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		source = args[0]
		args = args[1:]
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if source == "" {
		if len(fs.Args()) != 1 {
			return fmt.Errorf("usage: lazypost import <file> -dir <target> [-env <file>] [--format postman|insomnia] [--dry-run] [--force] [--strict]")
		}
		source = fs.Args()[0]
	} else if len(fs.Args()) > 0 {
		return fmt.Errorf("unexpected positional argument %q", fs.Args()[0])
	}
	if *target == "" {
		return fmt.Errorf("-dir is required")
	}

	result, err := importer.ParseFile(source, importer.ParseOptions{
		Format:           *format,
		EnvironmentFiles: environmentFiles,
	})
	if err != nil {
		return err
	}
	summary, err := app.ImportCollection(result, app.ImportOptions{
		Target: *target,
		Force:  *force,
		DryRun: *dryRun,
		Strict: *strict,
	})
	for _, warning := range summary.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", warning.String())
	}
	if err != nil {
		return err
	}
	verb := "imported"
	if *dryRun {
		verb = "would import"
	}
	fmt.Printf("%s %d folder(s), %d request(s), %d environment(s) into %s", verb, summary.Folders, summary.Requests, summary.Environments, summary.Target)
	if len(summary.Warnings) > 0 {
		fmt.Printf(" with %d warning(s)", len(summary.Warnings))
	}
	fmt.Println()
	return nil
}
