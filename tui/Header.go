/*
tui/Header.go

Implements the header model for the TUI.
Displays the target directory name, git information if
applicable, and the active panel indicator if applicable.
*/

package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/pkg/models"
	"github.com/go-git/go-git/v5"
)

// Msg containing the active panel index
type ActivePanelMsg struct {
	Panel int
}

// Command to return an ActivePanelMsg
func SetActivePanel(panel int) tea.Cmd {
	return func() tea.Msg {
		return ActivePanelMsg{Panel: panel}
	}
}

// Header data model
type headerModel struct {
	path          string
	directoryName string
	subPath       string
	isGitRepo     bool
	currentBranch string
	hash          string
	width         int
	activePanel   int
	gitFilter     *models.GitFilter
}

// Create a new header model
func newHeaderModel(absTarget string, absGitRoot string, gitFilter *models.GitFilter) headerModel {
	absPath, err := filepath.Abs(absTarget)
	if err != nil {
		absPath = absTarget
	}

	// Determine directory name for the box and sub-path
	var dirName string
	var subPath string

	if absGitRoot != "" {
		// Use the git repo root name for the box
		dirName = filepath.Base(absGitRoot)
		if rel, err := filepath.Rel(absGitRoot, absPath); err == nil && rel != "." {
			subPath = rel
		}
	} else {
		// use the target directory name
		dirName = filepath.Base(absPath)
	}

	if dirName == "." || dirName == "" {
		if wd, err := os.Getwd(); err == nil {
			dirName = filepath.Base(wd)
		} else {
			dirName = "."
		}
	}

	// Get the branch name and current commit hash
	var isGitRepo bool
	var currentBranch string
	var hash string
	if absGitRoot != "" {
		repo, err := git.PlainOpen(absGitRoot)
		if err == nil {
			headRef, err := repo.Head()
			if err == nil {
				currentBranch = headRef.Name().Short()
				hash = headRef.Hash().String()[0:7]
				isGitRepo = true
			}
		}
	}

	return headerModel{
		path:          absPath,
		directoryName: dirName,
		subPath:       subPath,
		isGitRepo:     isGitRepo,
		currentBranch: currentBranch,
		hash:          hash,
		activePanel:   0,
		gitFilter:     gitFilter,
	}
}

// Header update function
func (h *headerModel) Update(msg tea.Msg, width int) tea.Cmd {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = width // Update model with passed width if the window size message is sent
	case ActivePanelMsg:
		h.activePanel = m.Panel // Used to set active panel dot
	}

	return tea.Batch(cmds...)
}

func (h headerModel) View() string {
	// Create git information string
	gitString := ""
	if h.isGitRepo {
		gitString = gitStyle.String() + boldText.Render(h.currentBranch) + " "
		if h.hash != "" {
			gitString += hashStyle.Render(h.hash) + " "
		}

		// Add date filter information if present
		if h.gitFilter != nil {
			filterInfo := h.formatDateFilter()
			if filterInfo != "" {
				gitString += filterStyle.Render(filterInfo) + " "
			}
		}
	}

	// Create sub-path string if targeting a subdirectory
	subPathString := ""
	if h.subPath != "" {
		subPathString = subPathStyle.Render(" /" + h.subPath + " ")
	}

	// Create directory name with box
	dirBox := directoryStyle.Render(h.directoryName)
	preGitInfo := "──"

	// Create dot string
	dotString := " " + activeDot + " " + inactiveDot + " "
	if h.activePanel == 1 {
		dotString = " " + inactiveDot + " " + activeDot + " "
	}

	// Calculate the width of the components
	dirBoxWidth := lipgloss.Width(dirBox)
	subPathWidth := lipgloss.Width(subPathString)
	preGitInfoWidth := lipgloss.Width(preGitInfo)
	gitStringWidth := lipgloss.Width(gitString)
	dotStringWidth := lipgloss.Width(dotString)

	// calculate filler width
	var content string
	var fillerWidth int
	if h.width <= SINGLE_PANEL_WIDTH {
		fillerWidth = h.width - dirBoxWidth - subPathWidth - preGitInfoWidth - dotStringWidth
		content = dotString
	} else {
		fillerWidth = h.width - dirBoxWidth - subPathWidth - preGitInfoWidth - gitStringWidth
		content = gitString
	}

	// Add line to fill the space
	if fillerWidth < 0 {
		fillerWidth = 0
	}
	filler := strings.Repeat("─", fillerWidth)

	return lipgloss.JoinHorizontal(lipgloss.Center,
		dirBox,
		subPathString,
		preGitInfo,
		content,
		filler,
	)
}

// formatDateFilter creates a display string for active date filters
func (h headerModel) formatDateFilter() string {
	if h.gitFilter == nil {
		return ""
	}

	var parts []string

	if h.gitFilter.Since != nil {
		parts = append(parts, "since:"+h.gitFilter.Since.Format("2006-01-02"))
	}

	if h.gitFilter.Until != nil {
		parts = append(parts, "until:"+h.gitFilter.Until.Format("2006-01-02"))
	}

	if len(parts) == 0 {
		return ""
	}

	return "[" + strings.Join(parts, ", ") + "]"
}

// Styles for the header
var directoryStyle = lipgloss.NewStyle().
	Bold(true).
	Border(lipgloss.RoundedBorder()).
	Padding(0, 1)
var gitStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7")).
	SetString(" git: ")
var subPathStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))
var hashStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))
var filterStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("6"))
var boldText = lipgloss.NewStyle().Bold(true)
var activeDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
var inactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
