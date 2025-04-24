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
