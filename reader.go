// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yacr is yet another CSV reader (and writer) with small memory usage.
package yacr

import (
	"bytes"
	"encoding"
	"fmt"
	"github.com/harikb/bufio"
	"io"
	"reflect"
	"strconv"
)

// Reader provides an interface for reading CSV data
// (compatible with rfc4180 and extended with the option of having a separator other than ",").
// Successive calls to the Scan method will step through the 'fields', skipping the separator/newline between the fields.
// The EndOfRecord method tells when a field is terminated by a line break.
type Reader struct {
	*bufio.Scanner
	sep    byte // values separator
	quoted bool // specify if values may be quoted (when they contain separator or newline)
	guess  bool // try to guess separator based on the file header
	eor    bool // true when the most recent field has been terminated by a newline (not a separator).
	lineno int  // current line number (not record number)
	empty  bool // true when the current line is empty (or a line comment)

	Trim    bool // trim spaces (only on unquoted values). Break rfc4180 rule: "Spaces are considered part of a field and should not be ignored."
	Comment byte // character marking the start of a line comment. When specified, line comment appears as empty line.
	Lazy    bool // specify if quoted values may contains unescaped quote not followed by a separator or a newline
}

// DefaultReader creates a "standard" CSV reader (separator is comma and quoted mode active)
func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, ',', true, false)
}

// NewReader returns a new CSV scanner to read from r.
// When quoted is false, values must not contain a separator or newline.
func NewReader(r io.Reader, sep byte, quoted, guess bool) *Reader {
	s := &Reader{bufio.NewScanner(r), sep, quoted, guess, true, 1, false, false, 0, false}
	s.Split(s.ScanField)
	return s
}

// ScanRecord decodes one line fields to values.
// It's like fmt.Scan or database.sql.Rows.Scan.
func (s *Reader) ScanRecord(values ...interface{}) (int, error) {
	for i, value := range values {
		if !s.Scan() {
			return i, s.Err()
		}
		if i == 0 { // skip empty line (or line comment)
			for s.EmptyLine() {
				if !s.Scan() {
					return i, s.Err()
				}
			}
		}
		if err := s.value(value, true); err != nil {
			return i, err
		} else if s.EndOfRecord() != (i == len(values)-1) {
			return i, fmt.Errorf("unexpected number of fields: want %d, got %d (or more)", len(values), i+2)
		}
	}
	return len(values), nil
}

// ScanValue advances to the next token and decodes field's content to value.
// The value may point to data that will be overwritten by a subsequent call to Scan.
func (s *Reader) ScanValue(value interface{}) error {
	if !s.Scan() {
		return s.Err()
	}
	return s.value(value, false)
}

// Value decodes field's content to value.
// The value may point to data that will be overwritten by a subsequent call to Scan.
func (s *Reader) Value(value interface{}) error {
	return s.value(value, false)
}
func (s *Reader) value(value interface{}, copied bool) error {
	var err error
	switch value := value.(type) {
	case nil:
	case *string:
		*value = s.Text()
	case *int:
		*value, err = strconv.Atoi(s.Text())
	case *int32:
		var i int64
		i, err = strconv.ParseInt(s.Text(), 10, 32)
		*value = int32(i)
	case *int64:
		*value, err = strconv.ParseInt(s.Text(), 10, 64)
	case *bool:
		*value, err = strconv.ParseBool(s.Text())
	case *float64:
		*value, err = strconv.ParseFloat(s.Text(), 64)
	case *[]byte:
		if copied {
			v := s.Bytes()
			c := make([]byte, len(v))
			copy(c, v)
			*value = c
		} else {
			*value = s.Bytes()
		}
	case encoding.TextUnmarshaler:
		err = value.UnmarshalText(s.Bytes())
	default:
		return s.scanReflect(value)
	}
	return err
}

func (s *Reader) scanReflect(v interface{}) (err error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("unsupported type %T", v)
	}
	dv := reflect.Indirect(rv)
	switch dv.Kind() {
	case reflect.String:
		dv.SetString(s.Text())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		i, err = strconv.ParseInt(s.Text(), 10, dv.Type().Bits())
		if err == nil {
			dv.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var i uint64
		i, err = strconv.ParseUint(s.Text(), 10, dv.Type().Bits())
		if err == nil {
			dv.SetUint(i)
		}
	case reflect.Bool:
		var b bool
		b, err = strconv.ParseBool(s.Text())
		if err == nil {
			dv.SetBool(b)
		}
	case reflect.Float32, reflect.Float64:
		var f float64
		f, err = strconv.ParseFloat(s.Text(), dv.Type().Bits())
		if err == nil {
			dv.SetFloat(f)
		}
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return
}

// LineNumber returns current line number (not record number)
func (s *Reader) LineNumber() int {
	return s.lineno
}

// EndOfRecord returns true when the most recent field has been terminated by a newline (not a separator).
func (s *Reader) EndOfRecord() bool {
	return s.eor
}

// EmptyLine returns true when the current line is empty or a line comment.
func (s *Reader) EmptyLine() bool {
	return s.empty && s.eor
}

// Sep returns the values separator used/guessed
func (s *Reader) Sep() byte {
	return s.sep
}

// ScanField implements bufio.SplitFunc for CSV.
// Lexing is adapted from csv_read_one_field function in SQLite3 shell sources.
func (s *Reader) ScanField(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 && s.eor {
		return 0, nil, nil
	}
	if s.guess {
		s.guess = false
		if b := guess(data); b > 0 {
			s.sep = b
		}
	}
	if s.quoted && len(data) > 0 && data[0] == '"' { // quoted field (may contains separator, newline and escaped quote)
		s.empty = false
		startLineno := s.lineno
		escapedQuotes := 0
		strict := true
		var c, pc, ppc byte
		// Scan until the separator or newline following the closing quote (and ignore escaped quote)
		for i := 1; i < len(data); i++ {
			c = data[i]
			if c == '\n' {
				s.lineno++
			} else if c == '"' {
				if pc == c { // escaped quote
					pc = 0
					escapedQuotes++
					continue
				}
			}
			if pc == '"' && c == s.sep {
				s.eor = false
				return i + 1, unescapeQuotes(data[1:i-1], escapedQuotes, strict), nil
			} else if pc == '"' && c == '\n' {
				s.eor = true
				return i + 1, unescapeQuotes(data[1:i-1], escapedQuotes, strict), nil
			} else if c == '\n' && pc == '\r' && ppc == '"' {
				s.eor = true
				return i + 1, unescapeQuotes(data[1:i-2], escapedQuotes, strict), nil
			}
			if pc == '"' && c != '\r' {
				if s.Lazy {
					strict = false
				} else {
					return 0, nil, fmt.Errorf("unescaped %c character at line %d", pc, s.lineno)
				}
			}
			ppc = pc
			pc = c
		}
		if atEOF {
			if c == '"' {
				s.eor = true
				return len(data), unescapeQuotes(data[1:len(data)-1], escapedQuotes, strict), nil
			}
			// If we're at EOF, we have a non-terminated field.
			return 0, nil, fmt.Errorf("non-terminated quoted field at line %d", startLineno)
		}
	} else if s.eor && s.Comment != 0 && len(data) > 0 && data[0] == s.Comment { // line comment
		s.empty = true
		for i, c := range data {
			if c == '\n' {
				return i + 1, nil, nil
			}
		}
		if atEOF {
			return len(data), nil, nil
		}
	} else { // unquoted field
		// Scan until separator or newline, marking end of field.
		for i, c := range data {
			if c == s.sep {
				s.eor = false
				if s.Trim {
					return i + 1, trim(data[0:i]), nil
				}
				return i + 1, data[0:i], nil
			} else if c == '\n' {
				s.lineno++
				if i > 0 && data[i-1] == '\r' {
					s.empty = s.eor && i == 1 // FIXME empty & trim
					s.eor = true
					if s.Trim {
						return i + 1, trim(data[0 : i-1]), nil
					}
					return i + 1, data[0 : i-1], nil
				}
				s.empty = s.eor && i == 0 // FIXME empty & trim
				s.eor = true
				if s.Trim {
					return i + 1, trim(data[0:i]), nil
				}
				return i + 1, data[0:i], nil
			}
		}
		// If we're at EOF, we have a final field. Return it.
		if atEOF {
			s.empty = false
			s.eor = true
			if s.Trim {
				return len(data), trim(data), nil
			}
			return len(data), data, nil
		}
	}
	// Request more data.
	return 0, nil, nil
}

func unescapeQuotes(b []byte, count int, strict bool) []byte {
	if count == 0 {
		return b
	}
	for i, j := 0, 0; i < len(b); i, j = i+1, j+1 {
		b[j] = b[i]
		if b[i] == '"' && (strict || i < len(b)-1 && b[i+1] == '"') {
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
			count[b]++
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
