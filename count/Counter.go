/*
count/Counter.go

Contains the global state and functions for counting lines in files.
*/

package count

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-enry/go-enry/v2"
)

type FileCount struct {
	Filetype string
	Count    int
}

type ValidFile struct {
	Filetype string
	Path     string
}

type walkData struct {
	Counts     map[string]FileCount
	ValidFiles []ValidFile
	TotalLines int
}

type CountLinesReturn struct {
	Filetype string
	Count    int
}

// CountLines counts and returns the number of lines in a file.
// It also updates the global Counts map, TotalLines variable, and Files list.
func CountLines(filePath string, content []byte) (CountLinesReturn, error) {
	base := filepath.Base(filePath)
	dot := strings.Index(base, ".")
	if dot == -1 {
		// No extension, return 0
		return CountLinesReturn{}, nil
	}
	extension := base[dot:]

	// detect language, extension is fallback
	ftype := enry.GetLanguage(filepath.Base(filePath), content)
	if ftype == "" {
		ftype = extension
	}

	file, err := os.Open(filePath)
	if err != nil {
		return CountLinesReturn{}, err
	}
	defer file.Close()

	c, err := lineCounter(file)
	if err != nil {
		log.Println(err)
		return CountLinesReturn{}, err
	}

	return CountLinesReturn{
		Filetype: ftype,
		Count:    c,
	}, nil
}

// Reads from r in chunks and counts the number of lines.
func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
