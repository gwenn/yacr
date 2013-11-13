// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yacr is yet another CSV reader (and writer) with small memory usage.
package yacr

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Reader provides an interface for reading CSV data
// (compatible with rfc4180 and extended with the option of having a separator other than ",").
// Successive calls to the Scan method will step through the 'fields', skipping the separator/newline between the fields.
// The EndOfRecord method tells when a field is terminated by a line break.
type Reader struct {
	*bufio.Scanner
	sep    byte // values separator
	quoted bool // specify if values may be quoted (when they contains separator or newline)
	guess  bool // try to guess separator based on the file header
	eor    bool // true when the most recent field has been terminated by a newline (not a separator).
	line   int  // current line number (not record number)
}

// DefaultReader creates a "standard" CSV reader (separator is comma and quoted mode active)
func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, ',', true, false)
}

// NewReader returns a new CSV scanner to read from r.
func NewReader(r io.Reader, sep byte, quoted, guess bool) *Reader {
	s := &Reader{bufio.NewScanner(r), sep, quoted, guess, true, 1}
	s.Split(s.scanField)
	return s
}

// EndOfRecord returns true when the most recent field has been terminated by a newline (not a separator).
func (s *Reader) EndOfRecord() bool {
	return s.eor
}

// Lexing adapted from csv_read_one_field function in SQLite3 shell sources.
func (s *Reader) scanField(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if s.guess {
		s.guess = false
		if b := guess(data); b > 0 {
			s.sep = b
		}
	}
	shift := 0
	if !s.eor { // s.eor should be initialized to true to make this work.
		shift = 1
		data = data[shift:]
	}
	if s.quoted && len(data) > 0 && data[0] == '"' { // quoted field (may contains separator, newline and escaped quote)
		startLine := s.line
		escapedQuotes := 0
		var c, pc byte
		// Scan until the separator or newline following the closing quote (and ignore escaped quote)
		for i := 1; i < len(data); i++ {
			c = data[i]
			if c == '\n' {
				s.line++
			} else if c == '"' {
				if pc == c { // escaped quote
					pc = 0
					escapedQuotes++
					continue
				}
			}
			if pc == '"' && c == s.sep {
				s.eor = false
				return i + shift, unescapeQuotes(data[1:i-1], escapedQuotes), nil
			} else if pc == '"' && c == '\n' {
				s.eor = true
				return i + shift + 1, unescapeQuotes(data[1:i-1], escapedQuotes), nil
			} else if c == '\n' && pc == '\r' && i >= 2 && data[i-2] == '"' {
				s.eor = true
				return i + shift + 1, unescapeQuotes(data[1:i-2], escapedQuotes), nil
			}
			if pc == '"' && c != '\r' {
				return 0, nil, fmt.Errorf("unescaped %c character at line %d", pc, s.line)
			}
			pc = c
		}
		if atEOF {
			if c == '"' {
				s.eor = true
				return len(data) + shift, unescapeQuotes(data[1:len(data)-1], escapedQuotes), nil
			}
			// If we're at EOF, we have a non-terminated field.
			return 0, nil, fmt.Errorf("non-terminated quoted field at line %d", startLine)
		}
	} else { // non-quoted field
		// Scan until separator or newline, marking end of field.
		for i, c := range data {
			if c == s.sep {
				s.eor = false
				return i + shift, data[0:i], nil
			} else if c == '\n' {
				s.eor = true
				s.line++
				if i > 0 && data[i-1] == '\r' {
					return i + shift + 1, data[0 : i-1], nil
				}
				return i + shift + 1, data[0:i], nil
			}
		}
		// If we're at EOF, we have a final, non-empty field. Return it.
		if atEOF {
			s.eor = true
			return len(data) + shift, data, nil
		}
	}
	// Request more data.
	return 0, nil, nil
}

func unescapeQuotes(b []byte, count int) []byte {
	if count == 0 {
		return b
	}
	for i, j := 0, 0; i < len(b); i, j = i+1, j+1 {
		b[j] = b[i]
		if b[i] == '"' {
			i++
		}
	}
	return b[:len(b)-count]
}

func guess(data []byte) byte {
	seps := []byte{',', ';', '\t', '|', ':'}
	count := make(map[byte]uint)
	for _, b := range data {
		if bytes.IndexByte(seps, b) >= 0 {
			count[b] += 1
			/*} else if b == '\n' {
			break*/
		}
	}
	var max uint
	var sep byte
	for b, c := range count {
		if c > max {
			max = c
			sep = b
		}
	}
	return sep
}
