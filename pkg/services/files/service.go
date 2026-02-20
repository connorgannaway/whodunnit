/*
pkg/services/files/service.go

File analysis service that handles directory walking, file counting,
and filtering.
*/

package files

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/connorgannaway/whodunnit/pkg/filters"
	"github.com/connorgannaway/whodunnit/pkg/models"
	"github.com/go-enry/go-enry/v2"
)

type Service struct {
	rootPath string
}

type AnalysisResult struct {
	FileCounts                 map[string]models.FileCount
	FileCountsAlphabeticalKeys []string
	FileCountsSortedKeys       []string
	ValidFiles                 []models.FileInfo
	TotalLines                 int
}

type ErrorMessage struct {
	Err error
}

// Create a new file analysis service
func NewService(rootPath string) *Service {
	return &Service{
		rootPath: rootPath,
	}
}

// Entry for directory analysis
func (s *Service) AnalyzeDirectory(filter *models.FileFilter) tea.Cmd {
	return func() tea.Msg {
		result, err := s.walkAndAnalyze(filter)
		if err != nil {
			return ErrorMessage{Err: err}
		}
		return result
	}
}

// handles filter creation, directory walking, and analysis
func (s *Service) walkAndAnalyze(filter *models.FileFilter) (AnalysisResult, error) {
	// Create file filter
	fileFilter := s.createFileFilter(filter)
	
	result := AnalysisResult{
		FileCounts: make(map[string]models.FileCount),
		ValidFiles: []models.FileInfo{},
	}
	
	err := s.walkDirectory(s.rootPath, nil, fileFilter, &result)
	if err != nil {
		return result, err
	}
	
	// Sort results
	s.sortResults(&result)
	
	return result, nil
}

// recursively walk the directory tree
func (s *Service) walkDirectory(root string, parentIgnore IgnoreFunc, fileFilter *filters.FileFilter, result *AnalysisResult) error {
	// Load .gitignore for current directory
	currentIgnore, err := s.loadGitignore(root)
	if err != nil {
		return err
	}
	
	// Combine gitignore functions
	combinedIgnore := s.combineIgnoreFunctions(parentIgnore, currentIgnore)
	
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		entryPath := filepath.Join(root, entry.Name())
		
		// Check if should be ignored by .gitignore
		if combinedIgnore != nil && combinedIgnore(entryPath, entry.IsDir()) {
			continue
		}
		
		if entry.IsDir() {
			// Recursively process subdirectories
			if err := s.walkDirectory(entryPath, combinedIgnore, fileFilter, result); err != nil {
				return err
			}
		} else {
			// Process file
			if err := s.processFile(entryPath, fileFilter, result); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// process a single file
func (s *Service) processFile(filePath string, fileFilter *filters.FileFilter, result *AnalysisResult) error {
	// Check if path is a regular file (handles symlinks to directories)
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() {
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if fileFilter.IsIgnored(filePath, content) {
		return nil
	}

	lineCount, err := s.countLines(filePath, content)
	if err != nil {
		return err
	}
	
	// Detect language/file type
	filetype := s.detectFileType(filePath, content)
	if filetype == "" {
		return nil // Skip files without recognizable type
	}
	
	fileInfo := models.FileInfo{
		Path:      filePath,
		Filetype:  filetype,
		LineCount: lineCount,
		Size:      stat.Size(),
	}
	
	// Update aggregated counts
	if existing, exists := result.FileCounts[filetype]; exists {
		existing.Count += lineCount
		result.FileCounts[filetype] = existing
	} else {
		result.FileCounts[filetype] = models.FileCount{
			Filetype: filetype,
			Count:    lineCount,
		}
	}
	
	// Add to valid files list
	result.ValidFiles = append(result.ValidFiles, fileInfo)
	result.TotalLines += lineCount
	
	return nil
}

// count number of lines in file content
func (s *Service) countLines(filePath string, content []byte) (int, error) {
	if len(content) == 0 {
		return 0, nil
	}
	
	// Count newlines
	count := bytes.Count(content, []byte{'\n'})
	
	// If file doesn't end with newline, add 1
	if len(content) > 0 && content[len(content)-1] != '\n' {
		count++
	}
	
	return count, nil
}

// go-enry wrapper for detecting file type
func (s *Service) detectFileType(filePath string, content []byte) string {
	language := enry.GetLanguage(filepath.Base(filePath), content)
	if language != "" {
		return language
	}
	
	// Fallback to extension-based detection
	ext := filepath.Ext(filePath)
	if ext != "" {
		return ext
	}
	
	return ""
}

// Create a file filter based on configuration
func (s *Service) createFileFilter(config *models.FileFilter) *filters.FileFilter {
	if config == nil {
		config = &models.FileFilter{
			IgnoreDotFiles:       true,
			IgnoreConfigFiles:    true,
			IgnoreGeneratedFiles: true,
			IgnoreVendorFiles:    true,
		}
	}
	
	return filters.NewFileFilter(
		filters.WithDotFiles(config.IgnoreDotFiles),
		filters.WithConfigFiles(config.IgnoreConfigFiles),
		filters.WithGeneratedFiles(config.IgnoreGeneratedFiles),
		filters.WithVendorFiles(config.IgnoreVendorFiles),
	)
}

// Create sorted lists of keys for UI display
func (s *Service) sortResults(result *AnalysisResult) {
	// Create alphabetical keys
	for filetype := range result.FileCounts {
		result.FileCountsAlphabeticalKeys = append(result.FileCountsAlphabeticalKeys, filetype)
	}
	sort.Strings(result.FileCountsAlphabeticalKeys)
	
	// Create count-sorted keys
	result.FileCountsSortedKeys = make([]string, len(result.FileCountsAlphabeticalKeys))
	copy(result.FileCountsSortedKeys, result.FileCountsAlphabeticalKeys)
	sort.Slice(result.FileCountsSortedKeys, func(i, j int) bool {
		return result.FileCounts[result.FileCountsSortedKeys[i]].Count > result.FileCounts[result.FileCountsSortedKeys[j]].Count
	})
}

// Type for gitignore matching function
type IgnoreFunc func(path string, isDir bool) bool

// loads and parses .gitignore patterns
func (s *Service) loadGitignore(dir string) (IgnoreFunc, error) {
	// attempt to open .gitignore
	gitignorePath := filepath.Join(dir, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	
	// read patterns
	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	// return matching function
	return func(path string, isDir bool) bool {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return false
		}
		
		for _, pattern := range patterns {
			cleanPattern := pattern
			if strings.HasSuffix(pattern, "/") {
				if !isDir {
					continue
				}
				cleanPattern = strings.TrimSuffix(pattern, "/")
			}
			
			if matched, err := doublestar.PathMatch(cleanPattern, rel); err == nil && matched {
				return true
			}
		}
		return false
	}, nil
}

// handles calling multiple ignore functions for recursive directory walking
func (s *Service) combineIgnoreFunctions(parent, current IgnoreFunc) IgnoreFunc {
	return func(path string, isDir bool) bool {
		// Always ignore .git directories
		if isDir && filepath.Base(path) == ".git" {
			return true
		}
		
		if parent != nil && parent(path, isDir) {
			return true
		}
		
		if current != nil && current(path, isDir) {
			return true
		}
		
		return false
	}
}
