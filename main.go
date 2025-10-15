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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/pkg/models"
	"github.com/connorgannaway/whodunnit/pkg/services/export"
	"github.com/connorgannaway/whodunnit/pkg/services/files"
	"github.com/connorgannaway/whodunnit/pkg/services/git"
	"github.com/connorgannaway/whodunnit/tui"
)

var Version = "dev"

func init() {
	var BoldUnderline = lipgloss.NewStyle().Bold(true).Underline(true)

	// Override the default usage function with a custom message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n  %s [options] [repo]\n\n", BoldUnderline.Render("Usage:"), os.Args[0])

		fmt.Fprintln(os.Stderr, BoldUnderline.Render("Options:"))
		flag.PrintDefaults()

		fmt.Fprintf(os.Stderr, `
%s
  # scan all files, including configuration files
  whodunnit --withConfigFiles

  # scan all files, including configuration files, of the target directory and output to JSON
  whodunnit --withConfigFiles --json target/directory

  # export data as XML format
  whodunnit --xml

  # scan commits from 2024-01-01 onwards
  whodunnit --since 2024-01-01

  # scan commits between specific dates
  whodunnit --start 2024-01-01 --end 2024-12-31

  # combine date and file filters
  whodunnit --withConfigFiles --until 2024-06-01 --json


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
	xml := flag.Bool("xml", false, "write xml to stdout")
	startFlag := flag.String("start", "", "filter commits after this date (YYYY-MM-DD)")
	sinceFlag := flag.String("since", "", "filter commits after this date (YYYY-MM-DD) - alias for --start")
	endFlag := flag.String("end", "", "filter commits before this date (YYYY-MM-DD)")
	untilFlag := flag.String("until", "", "filter commits before this date (YYYY-MM-DD) - alias for --end")
	
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

	// Parse and validate date flags
	gitFilter, err := parseGitFilter(*startFlag, *sinceFlag, *endFlag, *untilFlag)
	if err != nil {
		log.Fatalf("date parsing error: %v", err)
	}
	
	// Set up the file filter config based on flags
	fileFilter := &models.FileFilter{
		IgnoreDotFiles:       !*df,
		IgnoreConfigFiles:    !*cf,
		IgnoreGeneratedFiles: !*gf,
		IgnoreVendorFiles:    !*vf,
	}

	// Validate export format flags - only one can be specified
	exportFormats := 0
	var selectedFormat models.ExportFormat
	if *json {
		exportFormats++
		selectedFormat = models.ExportFormatJSON
	}
	if *xml {
		exportFormats++
		selectedFormat = models.ExportFormatXML
	}
	
	if exportFormats > 1 {
		log.Fatal("only one export format can be specified (--json or --xml)")
	}

	// Run without TUI if any export flag is set
	if exportFormats > 0 {
		err := exportData(rootfs, fileFilter, gitFilter, selectedFormat)
		if err != nil {
			log.Fatalf("export failed: %v", err)
		}
		return
	}

	rootModel := tui.NewRootModel(rootfs, fileFilter, gitFilter)

	// Create TUI
	program := tea.NewProgram(
		rootModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run TUI
	_, err = program.Run()
	if err != nil {
		panic(err)
	}

}

// parse and validates date filter flags
func parseGitFilter(startFlag, sinceFlag, endFlag, untilFlag string) (*models.GitFilter, error) {
	gitFilter := &models.GitFilter{}
	
	// Parse start date (--start or --since)
	startDate := startFlag
	if startDate == "" {
		startDate = sinceFlag
	} else if sinceFlag != "" {
		return nil, fmt.Errorf("cannot specify both --start and --since flags")
	}
	
	if startDate != "" {
		parsed, err := parseDate(startDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start/since date '%s': %v", startDate, err)
		}
		gitFilter.Since = &parsed
	}
	
	// Parse end date (--end or --until)
	endDate := endFlag
	if endDate == "" {
		endDate = untilFlag
	} else if untilFlag != "" {
		return nil, fmt.Errorf("cannot specify both --end and --until flags")
	}
	
	if endDate != "" {
		parsed, err := parseDate(endDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end/until date '%s': %v", endDate, err)
		}
		gitFilter.Until = &parsed
	}
	
	// Validate date range
	if gitFilter.Since != nil && gitFilter.Until != nil && gitFilter.Since.After(*gitFilter.Until) {
		return nil, fmt.Errorf("start date cannot be after end date")
	}
	
	return gitFilter, nil
}

// parse and validates a date string in various common formats
func parseDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",           // YYYY-MM-DD
		"2006-01-02T15:04:05",  // YYYY-MM-DDTHH:MM:SS
		"2006-01-02 15:04:05",  // YYYY-MM-DD HH:MM:SS
		"01/02/2006",           // MM/DD/YYYY
		"2006/01/02",           // YYYY/MM/DD
		"02 Jan 2006",          // DD Mon YYYY
	}
	
	for _, format := range formats {
		if parsed, err := time.Parse(format, dateStr); err == nil {
			return parsed, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unsupported date format, try YYYY-MM-DD")
}

// exportData handles data export in the specified format without starting the TUI
func exportData(rootPath string, fileFilter *models.FileFilter, gitFilter *models.GitFilter, format models.ExportFormat) error {
	fileService := files.NewService(rootPath)
	
	// Perform file analysis
	cmd := fileService.AnalyzeDirectory(fileFilter)
	result := cmd()
	
	switch r := result.(type) {
	case files.AnalysisResult:
		// Create analysis result
		analysisResult := &models.AnalysisResult{
			FileCounts:                 r.FileCounts,
			FileCountsAlphabeticalKeys: r.FileCountsAlphabeticalKeys,
			FileCountsSortedKeys:       r.FileCountsSortedKeys,
			ValidFiles:                 r.ValidFiles,
			TotalLines:                 r.TotalLines,
			RepositoryPath:             rootPath,
			FileFilterUsed:             fileFilter,
			GitFilterUsed:              gitFilter,
			AnalysisDate:               time.Now(),
			AuthorStats:                make(map[string]*models.AuthorStats),
		}
		
		// Add git blame analysis if we have a git repository and valid files
		if len(r.ValidFiles) > 0 {
			gitService, err := git.NewService(rootPath)
			if err == nil {
				// Create a channel for status updates (not used in JSON mode)
				statusChan := make(chan tea.Msg, 100)
				
				cmd := gitService.BlameFiles(r.ValidFiles, gitFilter, statusChan)
				gitResult := cmd()
				
				close(statusChan)
				
				switch gr := gitResult.(type) {
				case git.ResultMessage:
					analysisResult.AuthorStats = gr.AuthorStats
					analysisResult.AuthorKeys = gr.SortedKeys
				case git.ErrorMessage:
					// Log the error but continue with file analysis only
					fmt.Fprintf(os.Stderr, "git blame analysis failed: %v", gr.Err)
				}
			}
		}
		
		// Export in the specified format
		exportService := export.NewService(analysisResult)
		return exportService.ExportData(format, os.Stdout)
	case files.ErrorMessage:
		return r.Err
	default:
		return fmt.Errorf("unexpected result type")
	}
}
