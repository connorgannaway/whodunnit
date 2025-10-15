/*
pkg/services/export/service.go

Handles exporting to different output formats.
Supports JSON, XML, and extensible for additional formats.
*/

package export

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/connorgannaway/whodunnit/pkg/models"
)

type Service struct{
	result *models.AnalysisResult
}

// Create a new export service
func NewService(result *models.AnalysisResult) *Service {
	return &Service{
		result: result,
	}
}

// Handles exporting results in the specified format
func (s *Service) ExportData(format models.ExportFormat, writer io.Writer) error {
	switch format {
	case models.ExportFormatJSON:
		return s.exportJSON(writer)
	case models.ExportFormatXML:
		return s.exportXML(writer)
	default:
		return fmt.Errorf("unsupported export format: %v", format)
	}
}

// exports data as JSON
func (s *Service) exportJSON(writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(s.createExportData())
}


// exports data as XML
func (s *Service) exportXML(writer io.Writer) error {
	data := s.createExportData()
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")
	
	if _, err := writer.Write([]byte(xml.Header)); err != nil {
		return err
	}
	
	return encoder.Encode(data)
}

// Structures the results for export
func (s *Service) createExportData() ExportData {
	// Calculate filtered lines (sum of all author stats if blame analysis was done)
	filteredLines := 0
	for _, stats := range s.result.AuthorStats {
		filteredLines += stats.TotalLines
	}
	
	data := ExportData{
		Metadata: ExportMetadata{
			RepositoryPath: s.result.RepositoryPath,
			AnalysisDate:   s.result.AnalysisDate.Format(time.RFC3339),
			TotalLines:     s.result.TotalLines,
			TotalFiles:     len(s.result.ValidFiles),
			FilteredLines:  filteredLines,
			Filters:        s.createFilterMetadata(),
		},
		FileTypes: make([]FileTypeExport, 0, len(s.result.FileCounts)),
		Authors:   make([]AuthorExport, 0, len(s.result.AuthorStats)),
		Files:     make([]FileExport, 0, len(s.result.ValidFiles)),
	}
	
	// Add file type data
	for _, filetype := range s.result.FileCountsSortedKeys {
		count := s.result.FileCounts[filetype]
		percentage := float64(count.Count) / float64(s.result.TotalLines) * 100

		data.FileTypes = append(data.FileTypes, FileTypeExport{
			Type:       filetype,
			LineCount:  count.Count,
			Percentage: percentage,
		})
	}
	
	// Add author data
	for _, author := range s.result.AuthorKeys {
		stats := s.result.AuthorStats[author]
		authorData := AuthorExport{
			Name:       stats.Author,
			Email:      stats.Email,
			TotalLines: stats.TotalLines,
			FileTypes:  make([]FileTypeContribution, 0, len(stats.LinesByType)),
		}
		
		for _, filetype := range stats.SortedCountsKeys {
			fc := stats.LinesByType[filetype]
			authorData.FileTypes = append(authorData.FileTypes, FileTypeContribution{
				Type:  fc.Filetype,
				Lines: fc.Count,
			})
		}
		
		data.Authors = append(data.Authors, authorData)
	}
	
	// Add file data
	for _, file := range s.result.ValidFiles {
		data.Files = append(data.Files, FileExport{
			Path:      file.Path,
			Type:      file.Filetype,
			LineCount: file.LineCount,
			Size:      file.Size,
		})
	}
	
	return data
}

// Convert filter metadata for export
func (s *Service) createFilterMetadata() *FilterMetadata {
	if s.result.GitFilterUsed == nil {
		return nil
	}

	gitFilter := s.result.GitFilterUsed
	var dateRange *DateRangeFilter
	
	// Add date range information if present
	if gitFilter.Since != nil || gitFilter.Until != nil {
		dateRange = &DateRangeFilter{}
		
		if gitFilter.Since != nil {
			since := gitFilter.Since.Format("2006-01-02")
			dateRange.Since = &since
		}
		
		if gitFilter.Until != nil {
			until := gitFilter.Until.Format("2006-01-02")
			dateRange.Until = &until
		}
	}
	
	// Only return filter metadata if we have actual filters
	if dateRange != nil {
		return &FilterMetadata{
			DateRange: dateRange,
		}
	}
	
	return nil
}

// ExportData represents the structured export format
type ExportData struct {
	XMLName   xml.Name         `json:"-" xml:"analysis"`
	Metadata  ExportMetadata   `json:"metadata" xml:"metadata"`
	FileTypes []FileTypeExport `json:"file_types" xml:"file_types>file_type"`
	Authors   []AuthorExport   `json:"authors,omitempty" xml:"authors>author,omitempty"`
	Files     []FileExport     `json:"files" xml:"files>file"`
}

// ExportMetadata contains analysis metadata
type ExportMetadata struct {
	RepositoryPath string                 `json:"repository_path" xml:"repository_path"`
	AnalysisDate   string                 `json:"analysis_date" xml:"analysis_date"`
	TotalLines     int                    `json:"total_lines" xml:"total_lines"`
	TotalFiles     int                    `json:"total_files" xml:"total_files"`
	FilteredLines  int                    `json:"filtered_lines,omitempty" xml:"filtered_lines,omitempty"`
	Filters        *FilterMetadata        `json:"filters,omitempty" xml:"filters,omitempty"`
}

// FilterMetadata contains information about applied filters
type FilterMetadata struct {
	DateRange *DateRangeFilter `json:"date_range,omitempty" xml:"date_range,omitempty"`
}

// DateRangeFilter contains date filtering information
type DateRangeFilter struct {
	Since *string `json:"since,omitempty" xml:"since,omitempty"`
	Until *string `json:"until,omitempty" xml:"until,omitempty"`
}

// FileTypeExport represents file type statistics
type FileTypeExport struct {
	Type       string  `json:"type" xml:"type,attr"`
	LineCount  int     `json:"line_count" xml:"line_count"`
	Percentage float64 `json:"percentage" xml:"percentage"`
}

// AuthorExport represents author statistics
type AuthorExport struct {
	Name       string                 `json:"name" xml:"name,attr"`
	Email      string                 `json:"email,omitempty" xml:"email,omitempty"`
	TotalLines int                    `json:"total_lines" xml:"total_lines"`
	FileTypes  []FileTypeContribution `json:"file_types" xml:"file_types>contribution"`
}

// FileTypeContribution represents an author's contribution to a file type
type FileTypeContribution struct {
	Type  string `json:"type" xml:"type,attr"`
	Lines int    `json:"lines" xml:"lines"`
}

// FileExport represents individual file information
type FileExport struct {
	Path      string `json:"path" xml:"path"`
	Type      string `json:"type" xml:"type"`
	LineCount int    `json:"line_count" xml:"line_count"`
	Size      int64  `json:"size" xml:"size"`
}
