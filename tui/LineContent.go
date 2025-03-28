package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-enry/go-enry/v2"

	"github.com/connorgannaway/whodunnit/count"
)

const FILETYPE_WIDTH int = 20
const COUNT_WIDTH int = 12
const CONTENT_TOTAL_WIDTH int = FILETYPE_WIDTH + COUNT_WIDTH

type lineContentModel struct {
	counts               map[string]count.FileCount
	sortedCountsKeyArray []string
	totalLines           int

	viewport viewport.Model
	ready    bool
}

func newLineContentModel() lineContentModel {
	return lineContentModel{
		counts:               make(map[string]count.FileCount),
		sortedCountsKeyArray: []string{},
	}
}

func (c lineContentModel) generateContent() string {
	var content string

	var vpWidth int
	if c.ready {
		vpWidth = c.viewport.Width
	} else {
		vpWidth = CONTENT_TOTAL_WIDTH
	}

	var filetypeColWidth int
	if vpWidth < CONTENT_TOTAL_WIDTH {
		filetypeColWidth = vpWidth - COUNT_WIDTH
		if filetypeColWidth < 0 {
			filetypeColWidth = 0
		}
	} else {
		filetypeColWidth = FILETYPE_WIDTH
	}

	totalLabel := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Width(filetypeColWidth).
		Bold(true).
		Render("Total:")
	totalCount := lipgloss.NewStyle().
		Align(lipgloss.Right).
		Width(COUNT_WIDTH).
		Bold(true).
		Render(strconv.Itoa(c.totalLines))
	line := totalLabel + totalCount
	if vpWidth > CONTENT_TOTAL_WIDTH {
		line = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, line)
	}
	content += line + "\n"

	for _, k := range c.sortedCountsKeyArray {
		v := c.counts[k]
		colorCode := enry.GetColor(v.Filetype)
		truncated := truncateString(v.Filetype, filetypeColWidth)
		filetypeStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCode)).
			Align(lipgloss.Left).
			Width(filetypeColWidth).
			Render(truncated)
		countStr := lipgloss.NewStyle().
			Align(lipgloss.Right).
			Width(COUNT_WIDTH).
			Render(strconv.Itoa(v.Count))
		line = filetypeStr + countStr
		if vpWidth > CONTENT_TOTAL_WIDTH {
			line = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, line)
		}
		content += line + "\n"
	}
	return content
}

func (c *lineContentModel) Update(msg tea.Msg, width, height int) tea.Cmd {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case count.WalkDoneMsg:
		c.counts = m.Counts
		c.sortedCountsKeyArray = m.SortedCountsKeyArray
		c.totalLines = m.TotalLines
		if c.ready {
			c.viewport.SetContent(c.generateContent())
		}
	case tea.WindowSizeMsg:
		if !c.ready {
			c.viewport = viewport.New(width, height)
			c.viewport.SetContent(c.generateContent())
			c.ready = true
		} else {
			c.viewport.Width = width
			c.viewport.Height = height
			c.viewport.SetContent(c.generateContent())
		}
	}

	var vpCmd tea.Cmd
	c.viewport, vpCmd = c.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return tea.Batch(cmds...)
}

func (c lineContentModel) View() string {
	if c.ready {
		return c.viewport.View()
	}
	return ""
}
