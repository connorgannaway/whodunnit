package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/count"
)

type control struct {
	key, desc string
}

type footerModel struct {
	width	 int
	controls  []control
	controlsLR []control
	separator string
	status    string
	showLR	bool
	spinner   spinner.Model
}

var footerBold = lipgloss.NewStyle().
	Foreground(lipgloss.Color("7")).Bold(true)
var footerText = lipgloss.NewStyle().
	Foreground(lipgloss.Color("8"))
var footerSeparator = lipgloss.NewStyle().
	Foreground(lipgloss.Color("8"))

func newFooterModel() footerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return footerModel{
		controls: []control{
			{key: "↑", desc: "Move Up"},
			{key: "↓", desc: "Move Down"},
			{key: "q", desc: "Quit"},
			//{key: "?", desc: "Info"},
		},
		controlsLR: []control{
			{key: "↑", desc: "Move Up"},
			{key: "↓", desc: "Move Down"},
			{key: "←/→", desc: "Switch Panels"},
			{key: "q", desc: "Quit"},
			//{key: "?", desc: "Info"},
		},
		separator: " | ",
		status:    "",
		spinner:   s,
	}
}

func (f *footerModel) Init() tea.Cmd {
	return f.spinner.Tick
}

func (f *footerModel) Update(msg tea.Msg, width int) tea.Cmd {
	var cmds []tea.Cmd
	switch m := msg.(type) {
	case count.BlameStatusMsg:
		f.status = "Blaming: " + m.Filepath
	case count.BlameDoneMsg:
		f.status = ""
	case spinner.TickMsg:
		var cmd tea.Cmd
		f.spinner, cmd = f.spinner.Update(msg)
		return cmd
	case tea.WindowSizeMsg:
		f.width = width
		if width <= SINGLE_PANEL_WIDTH {
			f.showLR = true
		} else {
			f.showLR = false
		}
	}

	if f.status != "" {
		cmds = append(cmds, f.spinner.Tick)
	}
	return tea.Batch(cmds...)
}

func (f footerModel) View() string {
	var s string
	var controlsArray []control
	if f.showLR {
		controlsArray = f.controlsLR
	} else {
		controlsArray = f.controls
	}

	for i, c := range controlsArray {
		s += footerBold.Render(c.key) + " " + footerText.Render(c.desc)
		if f.showLR && i == (len(controlsArray)/2 - 1) {
			s += "\n"
			continue
		}
		if i < len(controlsArray)-1 {
			s += footerSeparator.Render(f.separator)
		}
	}
	statusLine := ""
	if f.status != "" {
		statusLine = f.spinner.View() + " " + f.status
	}
	return lipgloss.PlaceHorizontal(f.width, lipgloss.Center, s) + "\n" + statusLine
}