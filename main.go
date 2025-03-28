package main

import (
	"flag"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/connorgannaway/whodunnit/tui"
)

func main() {
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

	p := tea.NewProgram(
		tui.NewRootModel(rootfs),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err = p.Run()
	if err != nil {
		panic(err)
	}
}
