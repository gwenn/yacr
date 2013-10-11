// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr

import (
	"bytes"
	"errors"
	"io"
)

var (
	ErrBufferFull = errors.New("buffer full")
)

// endOfLiner gives the position of end of line.
type endOfLiner interface {
	eol([]byte) int // eol returns the index/position of end of line in the specified slice or -1 if not present.
}

// lineReader implements buffering for an io.Reader object.
type lineReader struct {
	buf   []byte
	rd    io.Reader
	r, w  int
	max   int
	eoler endOfLiner
}

func newLineReader(rd io.Reader, size, max int, eoler endOfLiner) *lineReader {
	lineReader := &lineReader{
		buf:   make([]byte, size),
		rd:    rd,
		max:   max,
		eoler: eoler,
	}
	if lineReader.eoler == nil {
		lineReader.eoler = lineReader
	}
	return lineReader
}

func (b *lineReader) eol(buf []byte) int {
	return bytes.IndexByte(buf, '\n')
}

func (b *lineReader) readLine() ([]byte, error) {
	p := b.r
	for {
		// Look in buffer.
		if i := b.eoler.eol(b.buf[p:b.w]); i >= 0 {
			line := b.buf[p : p+i]
			b.r = p + i + 1
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			return line, nil
		}

		// Slide existing data to beginning.
		if b.r > 0 {
			copy(b.buf, b.buf[b.r:b.w])
			b.w -= b.r
			b.r = 0
		}
		p = b.w

		// grow buffer as needed
		if b.w+100 > len(b.buf) {
			newSize := len(b.buf) * 2
			if newSize > b.max {
				b.r = b.w
				return b.buf[:b.w], ErrBufferFull
			}
			buf := make([]byte, newSize)
			copy(buf, b.buf[:b.w])
			b.buf = buf
			// TODO shrink...
		}
		n, err := b.rd.Read(b.buf[b.w:])
		b.w += n
		if err != nil {
			b.r = b.w
			if b.w > 0 && err == io.EOF { // EOF == EOL
				return b.buf[:b.w], nil
			}
			return b.buf[:b.w], err
		}
	}
	panic("not reached") // Go 1.1 unreachable code
}
