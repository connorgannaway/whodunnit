package tui

const FILETYPE_WIDTH int = 20
const COUNT_WIDTH int = 12
const CONTENT_TOTAL_WIDTH int = FILETYPE_WIDTH + COUNT_WIDTH

type SortType int

const (
	SortTypeAlphabetical SortType = iota
	SortTypeCount
)

const (
	containerTopPadding    = 1
	containerBottomPadding = 0
	containerLeftPadding   = 2
	containerRightPadding  = 2
)

const SINGLE_PANEL_WIDTH = 50
