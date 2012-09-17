// The author disclaims copyright to this source code.

// Yet another CSV reader (and writer) with small memory usage.
package yacr

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
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
	b      *bufio.Reader
	rd     io.Reader
	buf    []byte
	values [][]byte
}

// DefaultReader creates a "standard" CSV reader
func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, COMMA, true)
}

// DefaultFileReader creates a "standard" CSV reader for the specified file.
func DefaultFileReader(filepath string) (*Reader, error) {
	return NewFileReader(filepath, COMMA, true)
}

// NewReaderBytes creates a CSV reader for the specified bytes.
func NewReaderBytes(b []byte, sep byte, quoted bool) *Reader {
	return NewReader(bytes.NewBuffer(b), sep, quoted)
}

// NewReaderString creates a CSV reader for the specified content.
func NewReaderString(s string, sep byte, quoted bool) *Reader {
	return NewReader(strings.NewReader(s), sep, quoted)
}

// NewReader creates a custom DSV reader
func NewReader(rd io.Reader, sep byte, quoted bool) *Reader {
	return &Reader{Sep: sep, Quoted: quoted, b: bufio.NewReader(rd), rd: rd, values: make([][]byte, 20)}
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
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	if r.Guess {
		r.guess(line)
	}
	if r.Quoted {
		start := 0
		values, isPrefix := r.scanLine(line, false)
		for isPrefix {
			start = copyValues(values, start)
			line, err := r.readLine()
			if err != nil {
				return nil, err
			}
			values, isPrefix = r.scanLine(line, true)
		}
		return values, nil
	}
	return r.split(line), nil
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

func (r *Reader) scanLine(line []byte, continuation bool) ([][]byte, bool) {
	start := 0
	var a [][]byte
	if continuation {
		a = r.values
	} else {
		a = r.values[:0]
	}
	quotedChunk := continuation
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
			if continuation {
				fixLastChunk(a, chunk)
				continuation = false
			} else {
				a = append(a, chunk)
			}
			start = i + 1
		}
	}
	if endQuotedChunk >= 0 {
		chunk = unescapeQuotes(line[start:endQuotedChunk], escapedQuotes)
	} else {
		chunk = unescapeQuotes(line[start:], escapedQuotes)
	}
	if continuation {
		fixLastChunk(a, chunk)
	} else {
		a = append(a, chunk)
	}
	r.values = a // if cap(a) != cap(r.values)
	return a, quotedChunk
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

func (r *Reader) readLine() ([]byte, error) {
	var buf, line []byte
	var err error
	isPrefix := true
	for isPrefix {
		line, isPrefix, err = r.b.ReadLine()
		if err != nil {
			return nil, err
		}
		if buf == nil {
			if !isPrefix {
				return line, nil
			}
			buf = r.buf[:0]
		}
		buf = append(buf, line...)
	}
	r.buf = buf // if cap(buf) != cap(r.buf)
	return buf, nil
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

func copyValues(row [][]byte, start int) int {
	var dup []byte
	for i := start; i < len(row); i++ {
		dup = make([]byte, len(row[i]))
		copy(dup, row[i])
		row[i] = dup
	}
	return len(row)
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
