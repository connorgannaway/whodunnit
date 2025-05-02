/*
counts/Ignore.go

Extendable ignore filter and wrapper functions around go-enry
to check if a file should be ignored based on its path or content.
*/

package count

import "github.com/go-enry/go-enry/v2"

type FilterFunc func(path string, content []byte) bool

type Ignorer struct {
	filters []FilterFunc
}

type IgnoreConfig struct {
	IgnoreDotFiles       bool
	IgnoreConfigFiles    bool
	IgnoreGeneratedFiles bool
	IgnoreVendorFiles    bool
}

func DefaultIgnoreConfig() IgnoreConfig {
	return IgnoreConfig{
		IgnoreDotFiles:       true,
		IgnoreConfigFiles:    true,
		IgnoreGeneratedFiles: true,
		IgnoreVendorFiles:    true,
	}
}

// Creates a new Ignorer with passed filter functions.
func NewIgnorer(opts ...FilterFunc) *Ignorer {
	ign := &Ignorer{
		filters: []FilterFunc{
			func(path string, content []byte) bool {
				return enry.IsBinary(content)
			},
		},
	}
	for _, opt := range opts {
		if opt != nil {
			ign.filters = append(ign.filters, opt)
		}
	}
	return ign
}

// Checks if a file should be ignored based on filter functions.
func (i *Ignorer) IsIgnored(path string, content []byte) bool {
	for _, f := range i.filters {
		if f(path, content) {
			return true
		}
	}
	return false
}

// -- Filter functions --

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
