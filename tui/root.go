/*
tui/root.go

Defines the root application model for the TUI. This contains all other models
and some global state. It handles the initialization of the application, the
updating of the models, and the rendering of the views.
It also handles the window size and content area calculations, as well as
the sorting of the content based on user input.

*/

package tui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/pkg/models"
	"github.com/connorgannaway/whodunnit/pkg/services/files"
	"github.com/connorgannaway/whodunnit/pkg/services/git"
	git_lib "github.com/go-git/go-git/v5"
)

// Root application model. Contains all other models and some global state.
type rootModel struct {
	errors []error

	// Services
	fileService *files.Service
	gitService  *git.Service

	// Models
	header       headerModel
	lineContent  lineContentModel
	blameContent blameContentModel
	footer       footerModel

	// Data
	analysisResult *models.AnalysisResult

	// Window and content size variables
	windowWidth   int
	leftWidth     int
	rightWidth    int
	contentHeight int
	headerHeight  int

	// Configuration
	activePanel   int
	sortBy        SortType
	fileFilter    *models.FileFilter
	gitFilter     *models.GitFilter
	statusChannel chan tea.Msg
}

func NewRootModel(rootfs string, fileFilter *models.FileFilter, gitFilter *models.GitFilter) *rootModel {
	// Set default file filter if none provided
	if fileFilter == nil {
		fileFilter = &models.FileFilter{
			IgnoreDotFiles:       true,
			IgnoreConfigFiles:    true,
			IgnoreGeneratedFiles: true,
			IgnoreVendorFiles:    true,
		}
	}

	r := &rootModel{}

	// Initialize services
	r.fileService = files.NewService(rootfs)
	gitService, err := git.NewService(rootfs)
	if err == nil {
		r.gitService = gitService
	}

	// Configuration + state
	r.errors = []error{}
	r.activePanel = 0
	r.sortBy = SortTypeAlphabetical
	r.fileFilter = fileFilter
	r.gitFilter = gitFilter
	r.statusChannel = make(chan tea.Msg, 100)

	// Initialize empty analysis result
	r.analysisResult = &models.AnalysisResult{
		FileCounts:   make(map[string]models.FileCount),
		AuthorStats:  make(map[string]*models.AuthorStats),
		ValidFiles:   []models.FileInfo{},
		RepositoryPath: rootfs,
	}

	// Initialize submodels
	r.header = newHeaderModel(rootfs, gitFilter)
	r.footer = newFooterModel()
	r.lineContent = newLineContentModel(
		&r.analysisResult.FileCounts,
		&r.analysisResult.FileCountsAlphabeticalKeys,
		&r.analysisResult.FileCountsSortedKeys,
		&r.analysisResult.TotalLines,
		&r.sortBy,
	)
	r.blameContent = newBlameContentModel(
		&r.analysisResult.AuthorStats,
		&r.analysisResult.AuthorKeys,
		&r.sortBy,
	)

	return r
}

// Wait for and return the next message from the status channel.
func (r *rootModel) subscribeStatus() tea.Cmd {
	return func() tea.Msg {
		return <-r.statusChannel
	}
}

// Ran on initialization. Kick off the file analysis
func (r *rootModel) Init() tea.Cmd {
	return r.fileService.AnalyzeDirectory(r.fileFilter)
}

func (r *rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle messages based on message type
	switch m := msg.(type) {
	case files.AnalysisResult:
		// Update file analysis data
		r.analysisResult.FileCounts = m.FileCounts
		r.analysisResult.FileCountsAlphabeticalKeys = m.FileCountsAlphabeticalKeys
		r.analysisResult.FileCountsSortedKeys = m.FileCountsSortedKeys
		r.analysisResult.ValidFiles = m.ValidFiles
		r.analysisResult.TotalLines = m.TotalLines

		// Start git analysis if service is available
		if r.gitService != nil {
			cmds = append(cmds, r.subscribeStatus(), r.gitService.BlameFiles(m.ValidFiles, r.gitFilter, r.statusChannel))
		}
	case files.ErrorMessage:
		r.errors = append(r.errors, m.Err)
	case git.ResultMessage:
		// Update git analysis data
		r.analysisResult.AuthorStats = m.AuthorStats
		r.analysisResult.AuthorKeys = m.SortedKeys
	case git.StatusMessage:
		// Git analysis status update - resubscribe for next message
		cmds = append(cmds, r.subscribeStatus())
	case git.ErrorMessage:
		if !errors.Is(m.Err, git_lib.ErrRepositoryNotExists) {
			r.errors = append(r.errors, m.Err)
		}
	case tea.KeyMsg:
		// Handle key events
		switch m.String() {
		case "ctrl+c", "q", "esc":
			return r, tea.Quit
		case "left", "right":
			//Switch between panels if the window is in single panel mode
			if r.windowWidth <= SINGLE_PANEL_WIDTH {
				if r.activePanel == 0 {
					r.activePanel = 1
					cmds = append(cmds, SetActivePanel(1))
				} else {
					r.activePanel = 0
					cmds = append(cmds, SetActivePanel(0))
				}
			}
		case "s":
			// Toggle global sort type
			if r.sortBy == SortTypeAlphabetical {
				r.sortBy = SortTypeCount
			} else {
				r.sortBy = SortTypeAlphabetical
			}

			// Update the sort type for both content models and re-render
			if r.lineContent.ready {
				r.lineContent.viewport.SetContent(r.lineContent.generateContent())
			}
			if r.blameContent.ready {
				r.blameContent.viewport.SetContent(r.blameContent.generateContent())
			}
		}
	case tea.WindowSizeMsg:
		// Update global window & content size variables
		availableWidth := m.Width - containerLeftPadding - containerRightPadding
		availableHeight := m.Height - containerTopPadding - containerBottomPadding

		hView := r.header.View()
		fView := r.footer.View()
		headerHeight := lipgloss.Height(hView)
		footerHeight := lipgloss.Height(fView)
		contentAreaHeight := availableHeight - headerHeight - footerHeight

		leftWidth := availableWidth / 2
		rightWidth := availableWidth - leftWidth

		r.windowWidth = availableWidth
		r.headerHeight = headerHeight
		r.contentHeight = contentAreaHeight
		r.leftWidth = leftWidth
		r.rightWidth = rightWidth
	}

	// forward the message to all models
	if r.windowWidth <= SINGLE_PANEL_WIDTH {
		cmds = append(cmds, r.lineContent.Update(msg, r.windowWidth, r.contentHeight))
		cmds = append(cmds, r.blameContent.Update(msg, r.windowWidth, r.contentHeight))
	} else {
		cmds = append(cmds, r.lineContent.Update(msg, r.leftWidth, r.contentHeight))
		cmds = append(cmds, r.blameContent.Update(msg, r.rightWidth, r.contentHeight))
	}
	cmds = append(cmds, r.footer.Update(msg, r.windowWidth))
	cmds = append(cmds, r.header.Update(msg, r.windowWidth))

	return r, tea.Batch(cmds...)
}

func (r *rootModel) View() string {
	// show error list if there are any
	if len(r.errors) > 0 {
		return fmt.Sprintf("Errors: %v", r.errors)
	}

	// Get the views from all models
	headerView := r.header.View()
	footerView := footerMargin.Render(r.footer.View())
	lineContentView := lineContentMargin.Render(r.lineContent.View())
	blameContentView := blameContentMargin.Render(r.blameContent.View())

	// If the window is too small, show only one content panel
	var contentRow string
	if r.windowWidth <= SINGLE_PANEL_WIDTH {
		if r.activePanel == 0 {
			contentRow = lineContentView
		} else {
			contentRow = blameContentView
		}
	} else {
		contentRow = lipgloss.JoinHorizontal(lipgloss.Top, lineContentView, blameContentView)
	}

	// Combine all views into a single string
	return containerStyle.Render(
		headerView + "\n" +
			contentRow + "\n" +
			footerView,
	)
}

// Some styles
var containerStyle = lipgloss.NewStyle().
	Padding(containerTopPadding, containerLeftPadding, containerBottomPadding, containerRightPadding)
var footerMargin = lipgloss.NewStyle().MarginTop(1)
var lineContentMargin = lipgloss.NewStyle().MarginRight(1)
var blameContentMargin = lipgloss.NewStyle().MarginLeft(1)
