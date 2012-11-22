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

// LineReader implements buffering for an io.Reader object.
type LineReader struct {
	buf  []byte
	rd   io.Reader
	r, w int
	max  int
}

func NewLineReader(rd io.Reader, size, max int) *LineReader {
	return &LineReader{
		buf: make([]byte, size),
		rd:  rd,
		max: max,
	}
}

func (b *LineReader) ReadLine() ([]byte, error) {
	p := b.r
	for {
		// Look in buffer.
		if i := bytes.IndexByte(b.buf[p:b.w], '\n'); i >= 0 {
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
	panic("not reached")
}
