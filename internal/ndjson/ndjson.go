package ndjson

import (
	"bufio"
	"io"
)

// Wrap transforms a newline-delimited JSON stream into a JSON array
// so it can be decoded directly into a slice.
func Wrap(r io.Reader) io.ReadCloser {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		line := 0
		wrap := false
		s := bufio.NewScanner(r)
		for s.Scan() {
			if line == 0 && s.Bytes()[0] != '[' {
				wrap = true
				pw.Write([]byte("["))
			}
			if line > 0 && wrap {
				pw.Write([]byte(","))
			}
			pw.Write(s.Bytes())
			line++
		}
		if wrap {
			pw.Write([]byte("]"))
		}
		if err := s.Err(); err != nil {
			pw.CloseWithError(err)
		}
	}()
	return pr
}
