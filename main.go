package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/connorgannaway/whodunnit/count"
	"github.com/connorgannaway/whodunnit/tui"
	"github.com/connorgannaway/whodunnit/tui/JsonExport"
)

func main() {
	df := flag.Bool("withDotFiles", false, "include dot files")
	cf := flag.Bool("withConfigFiles", false, "include config files")
	gf := flag.Bool("withGeneratedFiles", false, "include generated files")
	vf := flag.Bool("withVendorFiles", false, "include vendor files")
	json := flag.Bool("json", false, "write json to stdout")
	flag.Parse()

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
