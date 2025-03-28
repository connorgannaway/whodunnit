package count

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
)

type BlameCount struct {
	Author      string
	Count       int
	LinesByType map[string]*FileCount
}

var BlameCounts map[string]*BlameCount = make(map[string]*BlameCount)

var BlameStatusChannel = make(chan tea.Msg)

func BlameRepo(rootFs string) error {
	repo, err := git.PlainOpen(rootFs)
	if err != nil {
		return err
	}

	headRef, err := repo.Head()
	if err != nil {
		return err
	}

	commit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return err
	}

	for _, file := range Files {
		localizedPath, _ := strings.CutPrefix(file.Path, rootFs+"/")

		go func(path string) {
			BlameStatusChannel <- BlameStatusMsg{Filepath: path}
		}(localizedPath)

		blame, err := git.Blame(commit, localizedPath)
		if err != nil {
			continue
		}

		for _, line := range blame.Lines {
			if _, ok := BlameCounts[line.AuthorName]; !ok {
				BlameCounts[line.AuthorName] = &BlameCount{
					Author:      line.AuthorName,
					Count:       0,
					LinesByType: make(map[string]*FileCount),
				}
			}
			if _, ok := BlameCounts[line.AuthorName].LinesByType[file.Filetype]; !ok {
				BlameCounts[line.AuthorName].LinesByType[file.Filetype] = &FileCount{
					Filetype: file.Filetype,
					Count:    0,
				}
			}
			BlameCounts[line.AuthorName].Count++
			BlameCounts[line.AuthorName].LinesByType[file.Filetype].Count++
		}
	}

	return nil
}

func StartBlameRepo(rootFs string) tea.Cmd {
	return func() tea.Msg {
		err := BlameRepo(rootFs)
		if err != nil {
			return BlameErrorMsg{
				Error: err,
			}
		}

		var BlameCountKeys []string
		for k := range BlameCounts {
			BlameCountKeys = append(BlameCountKeys, k)
		}
		sort.Slice(BlameCountKeys, func(i, j int) bool {
			return BlameCounts[BlameCountKeys[i]].Count > BlameCounts[BlameCountKeys[j]].Count
		})

		return BlameDoneMsg{
			Counts: BlameCounts,
			SortedKeys: BlameCountKeys,
		}
	}
}