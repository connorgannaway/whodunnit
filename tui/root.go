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
	"github.com/connorgannaway/whodunnit/count"
	"github.com/go-git/go-git/v5"
)

// Root application model. Contains all other models and some global state.
type rootModel struct {
	header       headerModel
	lineContent  lineContentModel
	blameContent blameContentModel
	footer       footerModel
	errors       []error

	windowWidth   int
	leftWidth     int
	rightWidth    int
	contentHeight int
	headerHeight  int

	activePanel int

	sortBy           SortType
	fileIgnoreConfig count.IgnoreConfig
}

func NewRootModel(rootfs string, ign *count.IgnoreConfig) rootModel {
	var ignoreCfg count.IgnoreConfig
	if ign == nil {
		ignoreCfg = count.DefaultIgnoreConfig()
	} else {
		ignoreCfg = *ign
	}

	return rootModel{
		header:           newHeaderModel(rootfs),
		lineContent:      newLineContentModel(),
		blameContent:     newBlameContentModel(),
		footer:           newFooterModel(),
		errors:           []error{},
		activePanel:      0,
		sortBy:           SortTypeAlphabetical,
		fileIgnoreConfig: ignoreCfg,
	}
}

// Wait for and return the next message from BlameStatusChannel.
// tea.Cmds run as a goroutine, so we can block here and wait for the next message.
func subscribeBlameStatus() tea.Cmd {
	return func() tea.Msg {
		return <-count.BlameStatusChannel
	}
}

// Ran on initialization. Kick off the file walk
func (r rootModel) Init() tea.Cmd {
	return func() tea.Msg {
		return count.Walk(r.header.path, r.fileIgnoreConfig)
	}
}

func (r rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle messages based on message type
	switch m := msg.(type) {
	case count.WalkDoneMsg:
		cmds = append(cmds, subscribeBlameStatus(), count.StartBlameRepo(r.header.path))
	case count.WalkErrorMsg:
		r.errors = append(r.errors, m.Err)
	case count.BlameStatusMsg:
		// Must resubscribe to the channel to get the next message
		cmds = append(cmds, subscribeBlameStatus())
	case count.BlameErrorMsg:
		if !errors.Is(m.Err, git.ErrRepositoryNotExists) {
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
			r.lineContent.sortBy = r.sortBy
			if r.lineContent.ready {
				r.lineContent.viewport.SetContent(r.lineContent.generateContent())
			}
			r.blameContent.sortBy = r.sortBy
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

func (r rootModel) View() string {
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
