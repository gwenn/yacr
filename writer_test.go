// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr_test

import (
	"bytes"
	. "github.com/gwenn/yacr"
	"testing"
)

func writeRow(w *Writer, row []string) {
	for _, field := range row {
		if !w.Write([]byte(field)) {
			break
		}
	}
	w.EndOfRecord()
}

func TestWriter(t *testing.T) {
	out := &bytes.Buffer{}
	w := DefaultWriter(out)
	writeRow(w, []string{"a", "b,\n", "c\"d"})
	checkNoError(t, w.Err())
	w.Flush()
	checkNoError(t, w.Err())
	expected := "a,\"b,\n\",\"c\"\"d\"\n"
	line := out.String()
	if expected != line {
		t.Errorf("Expected '%s', got '%s'", expected, line)
	}
}
