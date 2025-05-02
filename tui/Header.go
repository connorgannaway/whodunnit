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
	isGitRepo     bool
	currentBranch string
	hash          string
	width         int
	activePanel   int
}

// Create a new header model
func newHeaderModel(path string) headerModel {
	// evaluate the path
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		absPath = cleanPath
	}

	// Get the directory name
	d := filepath.Base(absPath)
	if d == "." || d == "" {
		if wd, err := os.Getwd(); err == nil {
			d = filepath.Base(wd)
		} else {
			d = "."
		}
	}

	// Get the branch name and current commit hash
	var isGitRepo bool = false
	var currentBranch string = ""
	var hash string = ""
	repo, err := git.PlainOpen(path)
	if err == nil {
		headRef, err := repo.Head()
		if err == nil {
			currentBranch = headRef.Name().Short()
			hash = headRef.Hash().String()[0:7]
			isGitRepo = true
		}
	}

	return headerModel{
		path:          absPath,
		directoryName: d,
		isGitRepo:     isGitRepo,
		currentBranch: currentBranch,
		hash:          hash,
		activePanel:   0,
	}
}

// Header update function
func (h *headerModel) Update(msg tea.Msg, width int) tea.Cmd {
	var cmds []tea.Cmd

	// Header doesn't need many updates...
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
	preGitInfoWidth := lipgloss.Width(preGitInfo)
	gitStringWidth := lipgloss.Width(gitString)
	dotStringWidth := lipgloss.Width(dotString)

	// calculate filler width
	var content string
	var fillerWidth int
	if h.width <= SINGLE_PANEL_WIDTH {
		fillerWidth = h.width - dirBoxWidth - preGitInfoWidth - dotStringWidth
		content = dotString
	} else {
		fillerWidth = h.width - dirBoxWidth - preGitInfoWidth - gitStringWidth
		content = gitString
	}

	// Add line to fill the space
	if fillerWidth < 0 {
		fillerWidth = 0
	}
	filler := strings.Repeat("─", fillerWidth)

	// String everything together
	return lipgloss.JoinHorizontal(lipgloss.Center,
		dirBox,
		preGitInfo,
		content,
		filler,
	)
}

// Styles for the header
var directoryStyle = lipgloss.NewStyle().
	Bold(true).
	Border(lipgloss.RoundedBorder()).
	Padding(0, 1)
var gitStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7")).
	SetString(" git: ")
var hashStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7"))
var boldText = lipgloss.NewStyle().Bold(true)
var activeDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
var inactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")