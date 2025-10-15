/*
pkg/models/types.go

Types used across the application.
*/

package models

import "time"

// FileInfo represents information about a single file
type FileInfo struct {
	Path      string
	Filetype  string
	LineCount int
	Size      int64
}

// FileCount represents aggregated count data for a file type
type FileCount struct {
	Filetype string
	Count    int
}

// BlameInfo represents git blame information for a single line
type BlameInfo struct {
	AuthorName  string
	AuthorEmail string
	CommitHash  string
	CommitDate  time.Time
	LineNumber  int
}

// AuthorStats represents aggregated statistics for an author
type AuthorStats struct {
	Author      string
	Email       string
	TotalLines  int
	LinesByType map[string]FileCount
	
	// For UI sorting
	SortedAlphabeticalKeys []string
	SortedCountsKeys       []string
}

// GitFilter represents filtering options for git operations
type GitFilter struct {
	// Date filtering
	Since *time.Time
	Until *time.Time
}

// FileFilter represents filtering options for file processing
type FileFilter struct {
	IgnoreDotFiles       bool
	IgnoreConfigFiles    bool
	IgnoreGeneratedFiles bool
	IgnoreVendorFiles    bool
}

// AnalysisResult represents the complete analysis result
type AnalysisResult struct {
	// File analysis
	FileCounts                 map[string]FileCount
	FileCountsAlphabeticalKeys []string
	FileCountsSortedKeys       []string
	ValidFiles                 []FileInfo
	TotalLines                 int
	
	// Git analysis
	AuthorStats map[string]*AuthorStats
	AuthorKeys  []string
	
	// Metadata
	RepositoryPath string
	AnalysisDate   time.Time
	GitFilterUsed  *GitFilter
	FileFilterUsed *FileFilter
}

// SortType represents different sorting options
type SortType int

const (
	SortTypeAlphabetical SortType = iota
	SortTypeCount
	SortTypeDate
)

// ExportFormat represents different export formats
type ExportFormat int

const (
	ExportFormatJSON ExportFormat = iota
	ExportFormatXML
)