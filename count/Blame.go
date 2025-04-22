package count

import (
	"runtime"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
)

type BlameCount struct {
	Author      string
	Count       int
	LinesByType map[string]*FileCount

	SortedAlphabeticalKeys []string
	SortedCountsKeys      []string
}

type BlameJob struct {
	file  ValidFile
	index int
}

var (
	BlameCounts          = make(map[string]*BlameCount)
	blameCountsLocker    sync.Mutex
	BlameStatusChannel   = make(chan tea.Msg)
)


func BlameRepo(rootFs string) error {
	numWorkers := runtime.NumCPU()/2
	if numWorkers < 1 {
		numWorkers = 1
	}
	jobs := make(chan BlameJob)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	totalFileCount := len(Files)

	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()

			// Each worker opens its own repo/commit object
			repo, err := git.PlainOpen(rootFs)
			if err != nil {
				return 
			}
			headRef, err := repo.Head()
			if err != nil {
				return
			}
			commit, err := repo.CommitObject(headRef.Hash())
			if err != nil {
				return
			}

			for job := range jobs {
				file := job.file
				current := job.index
				localizedPath, _ := strings.CutPrefix(file.Path, rootFs+"/")

				// non‑blocking status update
				select {
				case BlameStatusChannel <- BlameStatusMsg{
					Filepath:    localizedPath,
					CurrentFile: current,
					TotalFiles:  totalFileCount,
				}:
				default:
				}

				blame, err := git.Blame(commit, localizedPath)
				if err != nil {
					continue 
				}

				// Update the shared counts
				blameCountsLocker.Lock()
				for _, line := range blame.Lines {
					bc, ok := BlameCounts[line.AuthorName]
					if !ok {
						bc = &BlameCount{
							Author:      line.AuthorName,
							LinesByType: make(map[string]*FileCount),
						}
						BlameCounts[line.AuthorName] = bc
					}
					if _, ok := bc.LinesByType[file.Filetype]; !ok {
						bc.LinesByType[file.Filetype] = &FileCount{
							Filetype: file.Filetype,
						}
					}
					bc.Count++
					bc.LinesByType[file.Filetype].Count++
				}
				blameCountsLocker.Unlock()
			}
		}()
	}

	// feed jobs
	for i, f := range Files {
		jobs <- BlameJob{file: f, index: i + 1}
	}
	close(jobs)
	wg.Wait()
	return nil
}

func StartBlameRepo(rootFs string) tea.Cmd {
	return func() tea.Msg {
		//Catch errors before creating workers
		repo, err := git.PlainOpen(rootFs)
		if err != nil {
			return BlameErrorMsg{Error: err}
		}
		headRef, err := repo.Head()
		if err != nil {
			return BlameErrorMsg{Error: err}
		}
		_, err = repo.CommitObject(headRef.Hash())
		if err != nil {
			return BlameErrorMsg{Error: err}
		}

		// Blame the repo
		if err := BlameRepo(rootFs); err != nil {
			return BlameErrorMsg{Error: err}
		}

		// Sort Contributors by count
		keys := make([]string, 0, len(BlameCounts))
		for k := range BlameCounts {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return BlameCounts[keys[i]].Count > BlameCounts[keys[j]].Count
		})

		// Create alphabetical and count sorted key arrays
		for _, k := range keys {
			bc := BlameCounts[k]
			var filetypeKeys []string
			for k := range bc.LinesByType {
				filetypeKeys = append(filetypeKeys, k)
			}
			sort.Strings(filetypeKeys)
			bc.SortedAlphabeticalKeys = filetypeKeys

			SortedCountsKeys := make([]string, len(filetypeKeys))
    		copy(SortedCountsKeys, filetypeKeys)
			sort.Slice(SortedCountsKeys, func(i, j int) bool {
				return bc.LinesByType[SortedCountsKeys[i]].Count > bc.LinesByType[SortedCountsKeys[j]].Count
			})
			bc.SortedCountsKeys = SortedCountsKeys
		}

		return BlameDoneMsg{Counts: BlameCounts, SortedKeys: keys}
	}
}