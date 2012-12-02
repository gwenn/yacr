// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Yet another CSV reader (and writer) with small memory usage.
package yacr

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const (
	COMMA     = ','
	SEMICOLON = ';'
	TAB       = '\t'
	PIPE      = '|'
	COLON     = ':'
)

var seps = []byte{COMMA, SEMICOLON, TAB, PIPE, COLON}

type Reader struct {
	Sep    byte // values separator
	Quoted bool // Specify if values may be quoted (when they contains separator or newline)
	Guess  bool // values separator is guessed from the content of the (first) line
	//trim	bool
	rd     io.Reader
	buf    *LineReader
	values [][]byte
}

type quotedEndOfLiner struct {
	reader        *Reader
	startOfValue  bool
	quotedValue   bool
	previousQuote int
}

// DefaultReader creates a "standard" CSV reader (separator is comma and quoted mode active)
func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, COMMA, true)
}

// DefaultFileReader creates a "standard" CSV reader for the specified file (separator is comma and quoted mode active).
func DefaultFileReader(filepath string) (*Reader, error) {
	return NewFileReader(filepath, COMMA, true)
}

// NewReaderBytes creates a CSV reader for the specified bytes.
func NewReaderBytes(b []byte, sep byte, quoted bool) *Reader {
	return NewReader(bytes.NewReader(b), sep, quoted)
}

// NewReaderString creates a CSV reader for the specified content.
func NewReaderString(s string, sep byte, quoted bool) *Reader {
	return NewReader(strings.NewReader(s), sep, quoted)
}

// NewReader creates a custom DSV reader
func NewReader(rd io.Reader, sep byte, quoted bool) *Reader {
	r := &Reader{Sep: sep, Quoted: quoted, rd: rd, buf: NewLineReader(rd, 4096, 8*4096, nil), values: make([][]byte, 20)}
	if quoted {
		r.buf.eoler = &quotedEndOfLiner{reader: r, startOfValue: true}
	}
	return r
}

// NewFileReader creates a custom DSV reader for the specified file.
func NewFileReader(filepath string, sep byte, quoted bool) (*Reader, error) {
	rd, err := zopen(filepath)
	if err != nil {
		return nil, err
	}
	return NewReader(rd, sep, quoted), nil
}

func (r *Reader) Close() error {
	c, ok := r.rd.(io.Closer)
	if ok {
		return c.Close()
	}
	return nil
}

// MustClose is like Close except it panics on error
func (r *Reader) MustClose() {
	err := r.Close()
	if err != nil {
		panic("yacr.MustClose error: " + err.Error())
	}
}

// ReadRow consumes a line returning its values.
// The returned values are only valid until the next call to ReadRow.
func (r *Reader) ReadRow() ([][]byte, error) { // TODO let the caller choose to reuse or not the same values: ReadRow(values [][]byte) ([][]byte, error)
	line, err := r.buf.ReadLine()
	if err != nil {
		return nil, err
	}
	if r.Guess {
		r.guess(line)
	}
	if r.Quoted {
		return r.scanLine(line)
	}
	return r.split(line), nil
}

func (q *quotedEndOfLiner) eol(buf []byte) int {
	for i, b := range buf {
		if q.startOfValue {
			q.startOfValue = false
			q.quotedValue = b == '"'
		} else if q.quotedValue && b == '"' {
			q.previousQuote++
		} else if q.previousQuote > 0 {
			if q.previousQuote%2 != 0 {
				q.quotedValue = false
			}
			q.previousQuote = 0
		}
		if !q.quotedValue {
			if b == '\n' {
				q.startOfValue = true
				return i
			} else if b == q.reader.Sep { // FIXME Guess
				q.startOfValue = true
			}
		}
	}
	return -1
}

// MustReadRow is like ReadRow except that it panics on error
func (r *Reader) MustReadRow() [][]byte {
	row, err := r.ReadRow()
	if err == io.EOF {
		return nil
	} else if err != nil {
		panic("yacr.MustReadRow error: " + err.Error())
	}
	return row
}

func (r *Reader) scanLine(line []byte) ([][]byte, error) {
	start := 0
	a := r.values[:0]
	quotedChunk := false
	endQuotedChunk := -1
	escapedQuotes := 0
	var chunk []byte
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			if quotedChunk {
				if i < (len(line)-1) && line[i+1] == '"' {
					escapedQuotes += 1
					i++
				} else {
					quotedChunk = false
					endQuotedChunk = i
				}
			} else if i == 0 || line[i-1] == r.Sep {
				quotedChunk = true
				start = i + 1
			}
		} else if line[i] == r.Sep && !quotedChunk {
			if endQuotedChunk >= 0 {
				chunk = unescapeQuotes(line[start:endQuotedChunk], escapedQuotes)
				escapedQuotes = 0
				endQuotedChunk = -1
			} else {
				chunk = line[start:i]
			}
			a = append(a, chunk)
			start = i + 1
		}
	}
	if endQuotedChunk >= 0 {
		chunk = unescapeQuotes(line[start:endQuotedChunk], escapedQuotes)
	} else {
		chunk = unescapeQuotes(line[start:], escapedQuotes)
	}
	a = append(a, chunk)
	r.values = a // if cap(a) != cap(r.values)
	if quotedChunk {
		return nil, fmt.Errorf("Partial line: %q", string(line))
	}
	return a, nil
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

func fixLastChunk(values [][]byte, continuation []byte) {
	prefix := values[len(values)-1]
	prefix = append(prefix, '\n') // TODO \r\n ?
	prefix = append(prefix, continuation...)
	values[len(values)-1] = prefix
}

func (r *Reader) split(line []byte) [][]byte {
	start := 0
	a := r.values[:0]
	for i := 0; i < len(line); i++ {
		if line[i] == r.Sep {
			a = append(a, line[start:i])
			start = i + 1
		}
	}
	a = append(a, line[start:])
	r.values = a // if cap(a) != cap(r.values)
	return a
}

func (r *Reader) guess(line []byte) {
	count := make(map[byte]uint)
	for _, b := range line {
		if bytes.IndexByte(seps, b) >= 0 {
			count[b] += 1
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
	if max > 0 {
		r.Sep = sep
		r.Guess = false
	}
}

// CSV writer
type Writer struct {
	Sep    byte // values separator
	Quoted bool // Specify if values should be quoted (when they contain a separator or a newline)
	//trim	bool
	b *bufio.Writer
}

// DefaultWriter creates a "standard" CSV writer (separator is comma and quoted mode active)
func DefaultWriter(wr io.Writer) *Writer {
	return NewWriter(wr, COMMA, true)
}

// NewWriter creates a custom DSV writer (separator and quoted mode specified by the caller)
func NewWriter(wr io.Writer, sep byte, quoted bool) *Writer {
	return &Writer{Sep: sep, Quoted: quoted, b: bufio.NewWriter(wr)}
}

// Write ensures that row values are quoted when needed.
func (w *Writer) Write(row []string) (err error) {
	for i, v := range row {
		if i > 0 {
			err = w.b.WriteByte(w.Sep)
			if err != nil {
				return
			}
		}
		err = w.write([]byte(v)) // TODO avoid copy?
		if err != nil {
			return
		}
	}
	err = w.b.WriteByte('\n') // TODO \r\n ?
	if err != nil {
		return
	}
	return
}

// WriteRow ensures that row values are quoted when needed.
func (w *Writer) WriteRow(row [][]byte) (err error) {
	for i, v := range row {
		if i > 0 {
			err = w.b.WriteByte(w.Sep)
			if err != nil {
				return
			}
		}
		err = w.write(v)
		if err != nil {
			return
		}
	}
	err = w.b.WriteByte('\n') // TODO \r\n ?
	if err != nil {
		return
	}
	return
}

// MustWriteRow is like WriteRow except that it panics on error
func (w *Writer) MustWriteRow(row [][]byte) {
	err := w.WriteRow(row)
	if err != nil {
		panic("yacr.MustWriteRow error: " + err.Error())
	}
}

// Flush ensures the writer's buffer is flushed.
func (w *Writer) Flush() error {
	return w.b.Flush()
}

// MustFlush is like Flush except that it panics on error
func (w *Writer) MustFlush() {
	err := w.Flush()
	if err != nil {
		panic("yacr.MustFlush error: " + err.Error())
	}
}

func (w *Writer) write(value []byte) (err error) {
	// In quoted mode, value is enclosed between quotes if it contains Sep, quote or \n.
	if w.Quoted {
		last := 0
		for i, c := range value {
			switch c {
			case '"', '\n', w.Sep:
			default:
				continue
			}
			if last == 0 {
				err = w.b.WriteByte('"')
				if err != nil {
					return
				}
			}
			_, err = w.b.Write(value[last:i])
			if err != nil {
				return
			}
			err = w.b.WriteByte(c)
			if err != nil {
				return
			}
			if c == '"' {
				err = w.b.WriteByte(c) // escaped with another double quote
				if err != nil {
					return
				}
			}
			last = i + 1
		}
		_, err = w.b.Write(value[last:])
		if err != nil {
			return
		}
		if last != 0 {
			err = w.b.WriteByte('"')
		}
	} else {
		_, err = w.b.Write(value)
	}
	return
}

func DeepCopy(row [][]byte) [][]byte {
	dup := make([][]byte, len(row))
	for i := 0; i < len(row); i++ {
		dup[i] = make([]byte, len(row[i]))
		copy(dup[i], row[i])
	}
	return dup
}

type zReadCloser struct {
	f  *os.File
	rd io.ReadCloser
}

// TODO Create golang bindings for zlib (gzopen) or libarchive?
// Check 'mime' package
func zopen(filepath string) (io.ReadCloser, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	var rd io.ReadCloser
	// TODO zip
	ext := path.Ext(f.Name())
	if ext == ".gz" {
		rd, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
	} else if ext == ".bz2" {
		rd = ioutil.NopCloser(bzip2.NewReader(f))
	} else {
		rd = f
	}
	return &zReadCloser{f, rd}, nil
}
func (z *zReadCloser) Read(b []byte) (n int, err error) {
	return z.rd.Read(b)
}
func (z *zReadCloser) Close() (err error) {
	err = z.rd.Close()
	if err != nil {
		return
	}
	return z.f.Close()
}
