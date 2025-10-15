/*
pkg/filters/file_filter.go

File filtering logic based on go-enry.
Supports multiple filtering criteria and can be easily extended.
*/

package filters

import (
	"github.com/go-enry/go-enry/v2"
)

// FilterFunc represents a function that filters files
type FilterFunc func(path string, content []byte) bool

// FileFilter handles file filtering with multiple criteria
type FileFilter struct {
	filters []FilterFunc
}

// Creates a new file filter with the provided filter functions
func NewFileFilter(filters ...FilterFunc) *FileFilter {
	ff := &FileFilter{
		filters: []FilterFunc{
			// Always filter binary files
			func(path string, content []byte) bool {
				return enry.IsBinary(content)
			},
		},
	}
	
	for _, filter := range filters {
		if filter != nil {
			ff.filters = append(ff.filters, filter)
		}
	}
	
	return ff
}

// Checks if a file should be ignored based on all configured filters
func (ff *FileFilter) IsIgnored(path string, content []byte) bool {
	for _, filter := range ff.filters {
		if filter(path, content) {
			return true
		}
	}
	return false
}

// Add a new filter function
func (ff *FileFilter) AddFilter(filter FilterFunc) {
	if filter != nil {
		ff.filters = append(ff.filters, filter)
	}
}

// Filter Functions

func WithDotFiles(enabled bool) FilterFunc {
	if !enabled {
		return nil
	}
	return func(path string, _ []byte) bool {
		return enry.IsDotFile(path)
	}
}

func WithConfigFiles(enabled bool) FilterFunc {
	if !enabled {
		return nil
	}
	return func(path string, _ []byte) bool {
		return enry.IsConfiguration(path)
	}
}

func WithGeneratedFiles(enabled bool) FilterFunc {
	if !enabled {
		return nil
	}
	return func(path string, content []byte) bool {
		return enry.IsGenerated(path, content)
	}
}

func WithVendorFiles(enabled bool) FilterFunc {
	if !enabled {
		return nil
	}
	return func(path string, _ []byte) bool {
		return enry.IsVendor(path)
	}
}

// Additional potential filters
// WithExtensions, WithSizeRange, WithCustomPattern, WithMinLines, WithMaxLines
