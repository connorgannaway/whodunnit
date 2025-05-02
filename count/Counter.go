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

// Global result variables
var Counts map[string]FileCount = make(map[string]FileCount)
var Files []ValidFile = make([]ValidFile, 0)
var TotalLines int


// CountLines counts and returns the number of lines in a file.
// It also updates the global Counts map, TotalLines variable, and Files list.
func CountLines(filePath string, content []byte) (int, error) {
	base := filepath.Base(filePath)
	dot := strings.Index(base, ".")
	if dot == -1 {
		// No extension, return 0
		return 0, nil
	}
	extension := base[dot:]

	// detect language, extension is fallback
	ftype := enry.GetLanguage(filepath.Base(filePath), content)
	if ftype == "" {
		ftype = extension
	}

	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	c, err := lineCounter(file)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	// Update counts for this file type
	if _, ok := Counts[ftype]; !ok {
		Counts[ftype] = FileCount{Filetype: ftype, Count: 0}
	}
	Counts[ftype] = FileCount{Filetype: ftype, Count: Counts[ftype].Count + c}

	// record the file
	Files = append(Files, ValidFile{Filetype: ftype, Path: filePath})

	// Update the total lines count
	TotalLines = TotalLines + c

	return c, nil
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
