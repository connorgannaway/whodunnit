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

var BoldUnderline = lipgloss.NewStyle().Bold(true).Underline(true)

func init() {
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

	rootfs := flag.Arg(0)
	if rootfs == "" {
		rootfs = "."
	}

	dir, err := os.Stat(rootfs)
	if err != nil {
		log.Fatal(err)
	}
	if !dir.IsDir() {
		log.Fatalf("%s is not a directory", rootfs)
	}

	filetypeIgnoreConfig := &count.IgnoreConfig{
		IgnoreDotFiles:       !*df,
		IgnoreConfigFiles:    !*cf,
		IgnoreGeneratedFiles: !*gf,
		IgnoreVendorFiles:    !*vf,
	}

	if *json {
		out, err := JsonExport.ExportJSON(rootfs, *filetypeIgnoreConfig)
		if err != nil {
			log.Fatalf("json export failed: %v", err)
		}
		fmt.Println(string(out))
		return
	}

	program := tea.NewProgram(
		tui.NewRootModel(rootfs, filetypeIgnoreConfig),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = program.Run()
	if err != nil {
		panic(err)
	}

}
