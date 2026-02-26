/*
pkg/services/git/service.go

Git interactions service for git blame analysis
*/

package git

import (
	"maps"
	"runtime"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/connorgannaway/whodunnit/pkg/models"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Service struct {
	gitRoot string // absolute path to the git repo root
	repo    *git.Repository
}

// StatusMessage for progress updates
type StatusMessage struct {
	Filepath    string
	CurrentFile int
	TotalFiles  int
}

// ResultMessage for completed analysis
type ResultMessage struct {
	AuthorStats map[string]*models.AuthorStats
	SortedKeys  []string
}

// ErrorMessage for git errors
type ErrorMessage struct {
	Err error
}

func NewService(gitRoot string) (*Service, error) {
	repo, err := git.PlainOpen(gitRoot)
	if err != nil {
		return nil, err
	}

	return &Service{
		gitRoot: gitRoot,
		repo:    repo,
	}, nil
}

// Analyze git blame for the provided file list with filtering
func (s *Service) BlameFiles(files []models.FileInfo, filter *models.GitFilter, statusChan chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		headRef, err := s.repo.Head()
		if err != nil {
			return ErrorMessage{Err: err}
		}

		commitHash := headRef.Hash()

		// Setup parallel processing
		numWorkers := runtime.NumCPU() / 2
		if numWorkers < 1 {
			numWorkers = 1
		}

		jobs := make(chan blameJob, len(files))
		var wg sync.WaitGroup

		// Shared result storage
		authorStats := make(map[string]*models.AuthorStats)
		var statsMutex sync.Mutex

		// Start workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go s.blameWorker(jobs, &wg, s.gitRoot, commitHash, authorStats, &statsMutex, statusChan, len(files), filter)
		}

		// Queue jobs
		for i, file := range files {
			jobs <- blameJob{file: file, index: i + 1}
		}
		close(jobs)

		wg.Wait()

		keys := s.sortAuthorStats(authorStats)

		return ResultMessage{
			AuthorStats: authorStats,
			SortedKeys:  keys,
		}
	}
}

// represents work for a single file
type blameJob struct {
	file  models.FileInfo
	index int
}

// processes blame jobs
func (s *Service) blameWorker(jobs <-chan blameJob, wg *sync.WaitGroup, gitRoot string, commitHash plumbing.Hash,
	authorStats map[string]*models.AuthorStats, statsMutex *sync.Mutex, statusChan chan tea.Msg, totalFiles int, filter *models.GitFilter) {

	defer wg.Done()

	// Each worker opens its own repository instance to avoid shared state/race conditions
	repo, err := git.PlainOpen(gitRoot)
	if err != nil {
		select {
		case statusChan <- ErrorMessage{Err: err}:
		default:
		}
		return
	}

	workerCommit, err := repo.CommitObject(commitHash)
	if err != nil {
		select {
		case statusChan <- ErrorMessage{Err: err}:
		default:
		}
		return
	}

	for job := range jobs {
		file := job.file
		localizedPath := strings.TrimPrefix(file.Path, s.gitRoot+"/")

		// Send status update to ui (non-blocking)
		select {
		case statusChan <- StatusMessage{
			Filepath:    localizedPath,
			CurrentFile: job.index,
			TotalFiles:  totalFiles,
		}:
		default:
		}

		blame, err := git.Blame(workerCommit, localizedPath)
		if err != nil {
			// Skip files that can't be blamed (untracked, not in commit, etc.)
			continue
		}

		// Aggregate per-file stats locally
		localAgg := make(map[string]*models.AuthorStats)
		for _, line := range blame.Lines {
			// Apply date filtering to blame lines if filter is provided
			if !s.shouldIncludeBlameLine(line, filter) {
				continue
			}

			stats, exists := localAgg[line.AuthorName]
			if !exists {
				stats = &models.AuthorStats{
					Author:      line.AuthorName,
					Email:       "", // Email not directly available in blame line
					LinesByType: make(map[string]models.FileCount),
				}
				localAgg[line.AuthorName] = stats
			}

			// Update file type stats
			fcount, exists := stats.LinesByType[file.Filetype]
			if !exists {
				fcount = models.FileCount{Filetype: file.Filetype}
			}
			fcount.Count++
			stats.LinesByType[file.Filetype] = fcount
			stats.TotalLines++
		}

		// Merge local aggregation into shared authorStats
		statsMutex.Lock()
		for author, localStats := range localAgg {
			// get stats for an author
			sharedStats, exists := authorStats[author]
			if !exists {
				// make a copy of localStats to store in shared map
				copied := &models.AuthorStats{
					Author:      localStats.Author,
					Email:       localStats.Email,
					TotalLines:  localStats.TotalLines,
					LinesByType: make(map[string]models.FileCount),
				}
				maps.Copy(copied.LinesByType, localStats.LinesByType)
				authorStats[author] = copied
				continue
			}

			// Merge author counts into existing stats
			sharedStats.TotalLines += localStats.TotalLines
			if sharedStats.LinesByType == nil {
				sharedStats.LinesByType = make(map[string]models.FileCount)
			}
			for ftype, fcount := range localStats.LinesByType {
				sharedFileCount, ok := sharedStats.LinesByType[ftype]
				if !ok {
					sharedFileCount = models.FileCount{Filetype: ftype, Count: 0}
				}
				sharedFileCount.Count += fcount.Count
				sharedStats.LinesByType[ftype] = sharedFileCount
			}
			authorStats[author] = sharedStats
		}
		statsMutex.Unlock()
	}
}

// Create sorted key lists for author statistics
func (s *Service) sortAuthorStats(authorStats map[string]*models.AuthorStats) []string {
	keys := make([]string, 0, len(authorStats))
	for author := range authorStats {
		keys = append(keys, author)
	}

	// Sort by total line count (descending)
	sort.Slice(keys, func(i, j int) bool {
		return authorStats[keys[i]].TotalLines > authorStats[keys[j]].TotalLines
	})

	// Create sorted keys for each author's file types
	for _, author := range keys {
		stats := authorStats[author]

		// Alphabetical keys
		var filetypeKeys []string
		for filetype := range stats.LinesByType {
			filetypeKeys = append(filetypeKeys, filetype)
		}
		sort.Strings(filetypeKeys)
		stats.SortedAlphabeticalKeys = filetypeKeys

		// Count-sorted keys
		sortedCountsKeys := make([]string, len(filetypeKeys))
		copy(sortedCountsKeys, filetypeKeys)
		sort.Slice(sortedCountsKeys, func(i, j int) bool {
			return stats.LinesByType[sortedCountsKeys[i]].Count > stats.LinesByType[sortedCountsKeys[j]].Count
		})
		stats.SortedCountsKeys = sortedCountsKeys
	}

	return keys
}

// shouldIncludeBlameLine checks if a blame line matches the filter criteria
func (s *Service) shouldIncludeBlameLine(line *git.Line, filter *models.GitFilter) bool {
	if filter == nil {
		return true
	}

	// Apply date filtering directly using the line's date
	if filter.Since != nil && line.Date.Before(*filter.Since) {
		return false
	}
	if filter.Until != nil && line.Date.After(*filter.Until) {
		return false
	}

	return true
}
