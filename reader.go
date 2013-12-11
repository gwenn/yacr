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
	empty  bool // true when the current line is empty (or a line comment)

	Trim    bool // trim spaces (only on not-quoted values). Break rfc4180 rule: "Spaces are considered part of a field and should not be ignored."
	Comment byte // character marking the start of a line comment. When specified, line comment appears as empty line.
}

// DefaultReader creates a "standard" CSV reader (separator is comma and quoted mode active)
func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, ',', true, false)
}

// NewReader returns a new CSV scanner to read from r.
func NewReader(r io.Reader, sep byte, quoted, guess bool) *Reader {
	s := &Reader{bufio.NewScanner(r), sep, quoted, guess, true, 1, false, false, 0}
	s.Split(s.scanField)
	return s
}

// EndOfRecord returns true when the most recent field has been terminated by a newline (not a separator).
func (s *Reader) EndOfRecord() bool {
	return s.eor
}

// EmptyLine returns true when the current line is empty or a line comment.
func (s *Reader) EmptyLine() bool {
	return s.empty && s.eor
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
		s.empty = false
		startLine := s.line
		escapedQuotes := 0
		var c, pc, ppc byte
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
			} else if c == '\n' && pc == '\r' && ppc == '"' {
				s.eor = true
				return i + shift + 1, unescapeQuotes(data[1:i-2], escapedQuotes), nil
			}
			if pc == '"' && c != '\r' {
				return 0, nil, fmt.Errorf("unescaped %c character at line %d", pc, s.line)
			}
			ppc = pc
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
	} else if s.eor && s.Comment != 0 && len(data) > 0 && data[0] == s.Comment { // line comment
		s.empty = true
		for i, c := range data {
			if c == '\n' {
				return i + shift + 1, nil, nil
			}
		}
		if atEOF {
			return len(data) + shift, nil, nil
		}
	} else { // non-quoted field
		// Scan until separator or newline, marking end of field.
		for i, c := range data {
			if c == s.sep {
				s.eor = false
				if s.Trim {
					return i + shift, trim(data[0:i]), nil
				}
				return i + shift, data[0:i], nil
			} else if c == '\n' {
				s.line++
				if i > 0 && data[i-1] == '\r' {
					s.empty = s.eor && i == 1
					s.eor = true
					if s.Trim {
						return i + shift + 1, trim(data[0 : i-1]), nil
					}
					return i + shift + 1, data[0 : i-1], nil
				}
				s.empty = s.eor && i == 0
				s.eor = true
				if s.Trim {
					return i + shift + 1, trim(data[0:i]), nil
				}
				return i + shift + 1, data[0:i], nil
			}
		}
		// If we're at EOF, we have a final, non-empty field. Return it.
		if atEOF {
			s.empty = false
			s.eor = true
			if s.Trim {
				l := len(data)
				return l + shift, trim(data), nil
			}
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

// bytes.TrimSpace may return nil...
func trim(s []byte) []byte {
	t := bytes.TrimSpace(s)
	if t == nil {
		return s[0:0]
	}
	return t
}
