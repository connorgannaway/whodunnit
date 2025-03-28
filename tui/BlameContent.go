package tui

import (
	"errors"
	"strconv"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/connorgannaway/whodunnit/count"
	"github.com/go-enry/go-enry/v2"
	"github.com/go-git/go-git/v5"
)

type blameContentModel struct {
	counts               map[string]*count.BlameCount
	sortedCountsKeyArray []string
	isGitRepo            bool

	viewport viewport.Model
	ready    bool
}

func newBlameContentModel() blameContentModel {
	return blameContentModel{
		isGitRepo: true,
	}
}

func (c blameContentModel) generateContent() string {
	var content string

	var vpWidth int
	if c.ready {
		vpWidth = c.viewport.Width
	} else {
		vpWidth = CONTENT_TOTAL_WIDTH
	}

	var authorColWidth int
	if vpWidth < CONTENT_TOTAL_WIDTH {
		authorColWidth = vpWidth - COUNT_WIDTH
		if authorColWidth < 0 {
			authorColWidth = 0
		}
	} else {
		authorColWidth = FILETYPE_WIDTH
	}

	if len(c.sortedCountsKeyArray) > 0 {
		for _, k := range c.sortedCountsKeyArray {
			authorStr := lipgloss.NewStyle().
				Align(lipgloss.Left).
				Bold(true).
				Width(authorColWidth).
				Render(truncateString(c.counts[k].Author, authorColWidth))
			totalStr := lipgloss.NewStyle().
				Align(lipgloss.Right).
				Width(COUNT_WIDTH).
				Render(strconv.Itoa(c.counts[k].Count))
			line := authorStr + totalStr
			if vpWidth > CONTENT_TOTAL_WIDTH {
				line = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, line)
			}
			content += line + "\n"

			for _, f := range c.counts[k].LinesByType {
				var filetypeColWidth int
				if vpWidth < CONTENT_TOTAL_WIDTH {
					filetypeColWidth = vpWidth - COUNT_WIDTH - 2
					if filetypeColWidth < 0 {
						filetypeColWidth = 0
					}
				} else {
					filetypeColWidth = FILETYPE_WIDTH - 2
				}
				colorCode := enry.GetColor(f.Filetype)
				truncatedFiletype := truncateString(f.Filetype, filetypeColWidth)
				filetypeStr := lipgloss.NewStyle().
					Foreground(lipgloss.Color(colorCode)).
					Align(lipgloss.Left).
					Width(filetypeColWidth).
					Render(truncatedFiletype)
				countStr := lipgloss.NewStyle().
					Align(lipgloss.Right).
					Width(COUNT_WIDTH).
					Render(strconv.Itoa(f.Count))
				line = "  " + filetypeStr + countStr
				if vpWidth > CONTENT_TOTAL_WIDTH {
					line = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, line)
				}
				content += line + "\n"
			}
			content += "\n"
		}
	} else {
		if c.isGitRepo {
			content = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, "Blaming...")
		} else {
			content = lipgloss.PlaceHorizontal(vpWidth, lipgloss.Center, "Run in a git repository to see blame information.")
		}
	}
	return content
}

func (c *blameContentModel) Update(msg tea.Msg, width, height int) tea.Cmd {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case count.BlameDoneMsg:
		c.counts = m.Counts
		c.sortedCountsKeyArray = m.SortedKeys
		if c.ready {
			c.viewport.SetContent(c.generateContent())
		}
	case count.BlameErrorMsg:
		if errors.Is(m.Error, git.ErrRepositoryNotExists) {
			c.isGitRepo = false
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

func (c blameContentModel) View() string {
	if c.ready {
		return c.viewport.View()
	}
	return ""
}
