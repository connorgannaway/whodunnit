package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-git/go-git/v5"
)

type ActivePanelMsg struct {
	Panel int
}

func SetActivePanel(panel int) tea.Cmd {
	return func() tea.Msg {
		return ActivePanelMsg{Panel: panel}
	}
}

type headerModel struct {
	path          string
	directoryName string
	isGitRepo     bool
	currentBranch string
	hash          string
	width         int
	activePanel   int
}

func newHeaderModel(path string) headerModel {
	cleanPath := filepath.Clean(path)

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		absPath = cleanPath
	}

	d := filepath.Base(absPath)
	if d == "." || d == "" {
		if wd, err := os.Getwd(); err == nil {
			d = filepath.Base(wd)
		} else {
			d = "."
		}
	}

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

func (h *headerModel) Update(msg tea.Msg, width int) tea.Cmd {
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = width
	case ActivePanelMsg:
		h.activePanel = m.Panel
	}

	return tea.Batch(cmds...)
}

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

func (h headerModel) View() string {
	gitString := ""
	if h.isGitRepo {
		gitString = gitStyle.String() + boldText.Render(h.currentBranch) + " "
		if h.hash != "" {
			gitString += hashStyle.Render(h.hash) + " "
		}
	}

	dirBox := directoryStyle.Render(h.directoryName)
	preGitInfo := "──"

	dotString := " " + activeDot + " " + inactiveDot + " "
	if h.activePanel == 1 {
		dotString = " " + inactiveDot + " " + activeDot + " "
	}

	dirBoxWidth := lipgloss.Width(dirBox)
	preGitInfoWidth := lipgloss.Width(preGitInfo)
	gitStringWidth := lipgloss.Width(gitString)
	dotStringWidth := lipgloss.Width(dotString)

	var content string
	var fillerWidth int
	if h.width <= SINGLE_PANEL_WIDTH {
		fillerWidth = h.width - dirBoxWidth - preGitInfoWidth - dotStringWidth
		content = dotString
	} else {
		fillerWidth = h.width - dirBoxWidth - preGitInfoWidth - gitStringWidth
		content = gitString
	}

	if fillerWidth < 0 {
		fillerWidth = 0
	}
	filler := strings.Repeat("─", fillerWidth)

	return lipgloss.JoinHorizontal(lipgloss.Center,
		dirBox,
		preGitInfo,
		content,
		filler,
	)
}
