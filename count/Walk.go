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

func walkDir(root string, parentIgnore IgnoreFunc) error {
	currentIgnore, err := loadGitignore(root)
	if err != nil {
		return err
	}

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
		if ignoreFn(entryPath, entry.IsDir()) {
			if entry.IsDir() {
				continue
			}
			continue
		}
		if entry.IsDir() {
			if err := walkDir(entryPath, ignoreFn); err != nil {
				return err
			}
		} else {
			CountLines(entryPath)
		}
	}

	return nil
}

func Walk(rootDir string) tea.Msg {
	if err := walkDir(rootDir, nil); err != nil {
		return WalkErrorMsg{Err: err}
	}

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
		Counts:               Counts,
		SortedAlphabeticalKeys: FileTypeKeys,
		SortedCountsKeys:     SortedCountsKeys,
		TotalLines:           TotalLines,
	}
}
