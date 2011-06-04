package yacr

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
)

type Reader struct {
	sep    byte
	quoted bool
	//trim	bool
	b      *bufio.Reader
	buf    []byte
	values [][]byte
}

func DefaultReader(rd io.Reader) *Reader {
	return NewReader(rd, ',', true)
}
func NewReaderBytes(b []byte, sep byte, quoted bool) *Reader {
	return NewReader(bytes.NewBuffer(b), sep, quoted)
}
func NewReaderString(s string, sep byte, quoted bool) *Reader {
	return NewReader(strings.NewReader(s), sep, quoted)
}
func NewReader(rd io.Reader, sep byte, quoted bool) *Reader {
	return &Reader{sep: sep, quoted: quoted, b: bufio.NewReader(rd), values: make([][]byte, 20)}
}

func (r *Reader) ReadRow() ([][]byte, os.Error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}
	if r.quoted {
		values, isPrefix := r.scanLine(line)
		if isPrefix {
			panic("Embedded new line not supported yet")
		}
		return values, nil
	}
	return r.split(line), nil
}

func (r *Reader) scanLine(line []byte) ([][]byte, bool) {
	start := 0
	a := r.values[0:0]
	quotedChunk := false
	endQuotedChunk := 0
	escapedQuotes := 0
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			if quotedChunk {
				if i < (len(line)-1) && line[i+1] == '"' {
					escapedQuotes += 1
				} else {
					quotedChunk = false
					endQuotedChunk = i
				}
			} else if i == 0 || line[i-1] == r.sep {
				quotedChunk = true
				start = i + 1
			}
		} else if line[i] == r.sep && !quotedChunk {
			if endQuotedChunk != 0 {
				a = append(a, unescapeQuotes(line[start:endQuotedChunk], escapedQuotes))
				escapedQuotes = 0
				endQuotedChunk = 0
			} else {
				a = append(a, line[start:i])
			}
			start = i + 1
		}
	}
	if endQuotedChunk != 0 {
		a = append(a, unescapeQuotes(line[start:endQuotedChunk], escapedQuotes))
	} else {
		a = append(a, unescapeQuotes(line[start:], escapedQuotes))
	}
	r.values = a // if cap(a) != cap(r.values)
	return a, quotedChunk
}

func unescapeQuotes(b []byte, count int) []byte {
	if count == 0 {
		return b
	}
	c := make([]byte, len(b)-count)
	for i, j := 0, 0; i < len(b); i, j = i+1, j+1 {
		c[j] = b[i]
		if b[i] == '"' {
			i++
		}
	}
	return c
}

func (r *Reader) readLine() ([]byte, os.Error) {
	var buf, line []byte
	var err os.Error
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
			buf = r.buf[0:0]
		}
		buf = append(buf, line...)
	}
	r.buf = buf // if cap(buf) != cap(r.buf)
	return buf, nil
}

func (r *Reader) split(line []byte) [][]byte {
	start := 0
	a := r.values[0:0]
	for i := 0; i < len(line); i++ {
		if line[i] == r.sep {
			a = append(a, line[start:i])
			start = i + 1
		}
	}
	a = append(a, line[start:])
	r.values = a // if cap(a) != cap(r.values)
	return a
}

type Writer struct {
	sep    byte
	quoted bool
	//trim	bool
	b *bufio.Writer
}

func DefaultWriter(wr io.Writer) *Writer {
	return NewWriter(wr, ',', true)
}
func NewWriter(wr io.Writer, sep byte, quoted bool) *Writer {
	// TODO
	if quoted {
		panic("Quoted mode not supported yet")
	}
	return &Writer{sep: sep, quoted: quoted, b: bufio.NewWriter(wr)}
}

func (w *Writer) WriteRow(row [][]byte) (err os.Error) {
	for i, v := range row {
		if i > 0 {
			err = w.b.WriteByte(w.sep)
			if err != nil {
				return
			}
		}
		err = w.write(v)
		if err != nil {
			return
		}
	}
	err = w.b.WriteByte('\n')
	if err != nil {
		return
	}
	return
}

func (w *Writer) Flush() os.Error {
	return w.b.Flush()
}

func (w *Writer) write(value []byte) (err os.Error) {
	_, err = w.b.Write(value)
	return
}
