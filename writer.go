// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr

import (
	"bufio"
	"io"
)

// CSV writer
type Writer struct {
	b      *bufio.Writer
	sep    byte  // values separator
	quoted bool  // specify if values should be quoted (when they contain a separator or a newline)
	sor    bool  // true at start of record
	err    error // sticky error.
}

// DefaultWriter creates a "standard" CSV writer (separator is comma and quoted mode active)
func DefaultWriter(wr io.Writer) *Writer {
	return NewWriter(wr, ',', true)
}

// NewWriter returns a new CSV writer.
func NewWriter(w io.Writer, sep byte, quoted bool) *Writer {
	return &Writer{b: bufio.NewWriter(w), sep: sep, quoted: quoted, sor: true}
}

// Write ensures that field is quoted when needed.
func (w *Writer) Write(field []byte) bool {
	if w.err != nil {
		return false
	}
	if !w.sor {
		w.setErr(w.b.WriteByte(w.sep))
	}
	// In quoted mode, field is enclosed between quotes if it contains sep, quote or \n.
	if w.quoted {
		last := 0
		for i, c := range field {
			switch c {
			case '"', '\n', w.sep:
			default:
				continue
			}
			if last == 0 {
				w.setErr(w.b.WriteByte('"'))
			}
			if _, err := w.b.Write(field[last:i]); err != nil {
				w.setErr(err)
			}
			w.setErr(w.b.WriteByte(c))
			if c == '"' {
				w.setErr(w.b.WriteByte(c)) // escaped with another double quote
			}
			last = i + 1
		}
		if _, err := w.b.Write(field[last:]); err != nil {
			w.setErr(err)
		}
		if last != 0 {
			w.setErr(w.b.WriteByte('"'))
		}
	} else {
		if _, err := w.b.Write(field); err != nil {
			w.setErr(err)
		}
	}
	w.sor = false
	return w.err == nil
}

// EndOfRecord tells when a line break must be inserted.
func (w *Writer) EndOfRecord() {
	w.setErr(w.b.WriteByte('\n')) // TODO \r\n ?
	w.sor = true
}

// Flush ensures the writer's buffer is flushed.
func (w *Writer) Flush() {
	w.setErr(w.b.Flush())
}

// Err returns the first error that was encountered by the Writer.
func (w *Writer) Err() error {
	return w.err
}

// setErr records the first error encountered.
func (w *Writer) setErr(err error) {
	if w.err == nil {
		w.err = err
	}
}
