/*
tui/LineContent.go

Implements the line content model for the TUI.
This model displays filetype line counts in the current
sort order. This is rendered in a viewport on the left side
of the TUI.
*/

package tui

import (
	"strconv"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-enry/go-enry/v2"

	"github.com/connorgannaway/whodunnit/count"
)

type lineContentModel struct {
	counts                 map[string]count.FileCount
	sortedAlphabeticalKeys []string
	sortedCountsKeys       []string
	sortBy                 SortType
	totalLines             int

	viewport viewport.Model
	ready    bool
}

func newLineContentModel() lineContentModel {
	return lineContentModel{
		counts:                 make(map[string]count.FileCount),
		sortedAlphabeticalKeys: []string{},
		sortedCountsKeys:       []string{},
		sortBy:                 SortTypeAlphabetical,
	}
}

func (c lineContentModel) GetSortType() SortType {
	return c.sortBy
}

func (c lineContentModel) generateContent() string {
	var content string

	var vpWidth int
	if c.ready {
		vpWidth = c.viewport.Width
	} else {
		vpWidth = CONTENT_TOTAL_WIDTH
	}

	// Calculate width of filenames
	var filetypeColWidth int
	if vpWidth < CONTENT_TOTAL_WIDTH {
		filetypeColWidth = vpWidth - COUNT_WIDTH
		if filetypeColWidth < 0 {
			filetypeColWidth = 0
		}
	} else {
		filetypeColWidth = FILETYPE_WIDTH
	}

	// Generate total lines header
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

	// Set keys based on current sort type
	var keys []string
	if c.sortBy == SortTypeCount {
		keys = c.sortedCountsKeys
	} else {
		keys = c.sortedAlphabeticalKeys
	}

	// Iterate over every filetype, rendering the filetype and count
	for _, k := range keys {
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
		c.sortedAlphabeticalKeys = m.SortedAlphabeticalKeys
		c.sortedCountsKeys = m.SortedCountsKeys
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

	// Also pass the message to the viewport
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
