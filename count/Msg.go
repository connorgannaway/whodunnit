package count

type WalkDoneMsg struct {
	Counts               map[string]FileCount
	SortedCountsKeyArray []string
	TotalLines           int
}

type WalkErrorMsg struct {
	Err error
}

type BlameDoneMsg struct {
	Counts     map[string]*BlameCount
	SortedKeys []string
}

type BlameErrorMsg struct {
	Error error
}

type BlameStatusMsg struct {
	CurrentFile int
	TotalFiles  int
	Filepath string
}
