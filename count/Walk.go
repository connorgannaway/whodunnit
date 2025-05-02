/*
count/Walk.go

Functionality to walk a directory tree while respecting .gitignore files
and other file exclusion rules set by the user. Exposes a bubbletea-compatible
Walk function. Counting logic is delegated to count/Counter.CountLines.
*/

package count

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	tea "github.com/charmbracelet/bubbletea"
)

type IgnoreFunc func(path string, isDir bool) bool

func loadGitignore(dir string) (IgnoreFunc, error) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	f, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	// Read in gitignore patterns
	var patterns []string
	scanner := bufio.NewScanner(f)
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

	// Return a function that checks if a path matches any of the patterns
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

func walkDir(root string, parentIgnore IgnoreFunc, fileExclusions *Ignorer) error {
	currentIgnore, err := loadGitignore(root)
	if err != nil {
		return err
	}

	// Combine parent and current ignore functions, along with ignoring .git directories
	ignoreFn := func(path string, isDir bool) bool {
		if isDir && filepath.Base(path) == ".git" {
			return true
		}
		if parentIgnore != nil && parentIgnore(path, isDir) {
			return true
		}
		if currentIgnore != nil && currentIgnore(path, isDir) {
			return true
		}
		return false
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(root, entry.Name())

		// check if the entry should be ignored based on gitignores
		if ignoreFn(entryPath, entry.IsDir()) {
			continue
		}

		// Recursively walk directories
		if entry.IsDir() {
			if err := walkDir(entryPath, ignoreFn, fileExclusions); err != nil {
				return err
			}
		} else {
			// Load file content for filetype detection
			content, err := os.ReadFile(entryPath)
			if err != nil {
				return err
			}
			// Check if the file should be ignored based on app config
			if fileExclusions.IsIgnored(entryPath, content) {
				continue
			}
			// Pass the file to count/Counter.CountLines for handling 
			// filetype detection and line counting
			CountLines(entryPath, content)
		}
	}

	return nil
}

func Walk(rootDir string, filetypeExclusionConfig IgnoreConfig) tea.Msg {
	// Create ignorer from exclusion config
	fileExclusions := NewIgnorer(
		WithDotFiles(filetypeExclusionConfig.IgnoreDotFiles),
		WithConfigFiles(filetypeExclusionConfig.IgnoreConfigFiles),
		WithGeneratedFiles(filetypeExclusionConfig.IgnoreGeneratedFiles),
		WithVendorFiles(filetypeExclusionConfig.IgnoreVendorFiles),
	)

	if err := walkDir(rootDir, nil, fileExclusions); err != nil {
		return WalkErrorMsg{Err: err}
	}

	// Create alphabetical and count sorted key lists
	var FileTypeKeys []string
	for k := range Counts {
		FileTypeKeys = append(FileTypeKeys, k)
	}
	sort.Strings(FileTypeKeys)

	SortedCountsKeys := make([]string, len(FileTypeKeys))
	copy(SortedCountsKeys, FileTypeKeys)
	sort.Slice(SortedCountsKeys, func(i, j int) bool {
		return Counts[SortedCountsKeys[i]].Count > Counts[SortedCountsKeys[j]].Count
	})

	return WalkDoneMsg{
		Counts:                 Counts,
		SortedAlphabeticalKeys: FileTypeKeys,
		SortedCountsKeys:       SortedCountsKeys,
		TotalLines:             TotalLines,
	}
}
