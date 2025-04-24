package count

type WalkDoneMsg struct {
	Counts                 map[string]FileCount
	SortedAlphabeticalKeys []string
	SortedCountsKeys       []string
	TotalLines             int
}

type WalkErrorMsg struct {
	Err error
}

type BlameDoneMsg struct {
	Counts     map[string]*BlameCount
	SortedKeys []string
}

type BlameErrorMsg struct {
	Err error
}

type BlameStatusMsg struct {
	CurrentFile int
	TotalFiles  int
	Filepath    string
}

type WalkStatusMsg struct {
	Message string
}
