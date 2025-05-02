/*
main.go

Main entry point for whodunnit. Handles command-line arguments and
sets up the TUI or JSON export.
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/count"
	"github.com/connorgannaway/whodunnit/tui"
	"github.com/connorgannaway/whodunnit/tui/JsonExport"
)


var Version = "dev"

func init() {
	var BoldUnderline = lipgloss.NewStyle().Bold(true).Underline(true)

	// Override the default usage function with a custom message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n  %s [options] [repo]\n\n",BoldUnderline.Render("Usage:"), os.Args[0])

		fmt.Fprintln(os.Stderr, BoldUnderline.Render("Options:"))
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, `
%s
  # scan all files, including configuration files
  whodunnit --withConfigFiles

  # scan all files, including configuration files, of the target directory and output to JSON
  whodunnit --withConfigFiles --json repos/target

For more information, see https://github.com/connorgannaway/whodunnit.
`, BoldUnderline.Render("Examples:"))
	}
}


func main() {
	// Define and parse command-line flags
	df := flag.Bool("withDotFiles", false, "include dot files")
	cf := flag.Bool("withConfigFiles", false, "include config files")
	gf := flag.Bool("withGeneratedFiles", false, "include generated files")
	vf := flag.Bool("withVendorFiles", false, "include vendor files")
	verf := flag.Bool("version", false, "print version")
	json := flag.Bool("json", false, "write json to stdout")
	flag.Parse()

	if *verf {
		fmt.Printf("Version: %s\n", Version)
		return
	}

	// Grab directory from first arg
	rootfs := flag.Arg(0)
	if rootfs == "" {
		rootfs = "."
	}

	// ensure directory and not file
	dir, err := os.Stat(rootfs)
	if err != nil {
		log.Fatal(err)
	}
	if !dir.IsDir() {
		log.Fatalf("%s is not a directory", rootfs)
	}

	// Set up the ignore config based on flags
	filetypeIgnoreConfig := &count.IgnoreConfig{
		IgnoreDotFiles:       !*df,
		IgnoreConfigFiles:    !*cf,
		IgnoreGeneratedFiles: !*gf,
		IgnoreVendorFiles:    !*vf,
	}

	// Run without TUI if --json flag is set
	if *json {
		out, err := JsonExport.ExportJSON(rootfs, *filetypeIgnoreConfig)
		if err != nil {
			log.Fatalf("json export failed: %v", err)
		}
		fmt.Println(string(out))
		return
	}

	// Create TUI
	program := tea.NewProgram(
		tui.NewRootModel(rootfs, filetypeIgnoreConfig),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run TUI
	_, err = program.Run()
	if err != nil {
		panic(err)
	}

}
