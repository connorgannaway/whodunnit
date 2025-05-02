/*
tui/JsonExport/JsonExport.go

JsonExport provides functionality to export the data collected by the
application in JSON format separate to the TUI. It drives the file walk
and blame process and assembles the data into a JSON structure.
*/

package JsonExport

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/connorgannaway/whodunnit/count"
	"github.com/go-git/go-git/v5"
)

type jsonExportBody struct {
	IgnoredFileTypes count.IgnoreConfig
	TotalLines       int
	IncludedFiles    []count.ValidFile
	FileCounts       map[string]count.FileCount
	Blame            map[string]*count.BlameCount
}

// ExportJSON returns a JSON representation of the data collected by the
// application. It handles the file walk and blame process
func ExportJSON(rootfs string, cfg count.IgnoreConfig) ([]byte, error) {
	walkMsg := count.Walk(rootfs, cfg)
	if errMsg, ok := walkMsg.(count.WalkErrorMsg); ok {
		return nil, fmt.Errorf("walk error: %w", errMsg.Err)
	}

	blameMsg := count.StartBlameRepo(rootfs)()
	if errMsg, ok := blameMsg.(count.BlameErrorMsg); ok {
		if !errors.Is(errMsg.Err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("blame error: %w", errMsg.Err)
		}
	}

	body := jsonExportBody{
		IgnoredFileTypes: cfg,
		IncludedFiles:    count.Files,
		TotalLines:       count.TotalLines,
		FileCounts:       count.Counts,
		Blame:            count.BlameCounts,
	}
	return json.Marshal(body)
}
