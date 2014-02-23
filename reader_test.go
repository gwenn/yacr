// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr_test

import (
	"reflect"
	"strings"
	"testing"
	. "github.com/gwenn/yacr"
)

func makeReader(s string, quoted bool) *Reader {
	return NewReader(strings.NewReader(s), ',', quoted, false)
}

func readRow(r *Reader) []string {
	row := make([]string, 0, 10)
	for r.Scan() {
		if r.EmptyLine() { // skip empty line (or line comment)
			continue
		}
		row = append(row, r.Text())
		if r.EndOfRecord() {
			break
		}
	}
	return row
}

func checkValueCount(t *testing.T, expected int, values []string) {
	if len(values) != expected {
		t.Errorf("got %d value(s) (%#v); want %d", len(values), values, expected)
	}
}

func checkNoError(t *testing.T, e error) {
	if e != nil {
		t.Fatal(e)
	}
}

func checkEquals(t *testing.T, expected, actual []string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("got %#v; want %#v", actual, expected)
	}
}

func TestSingleValue(t *testing.T) {
	expected := "Foo"
	r := makeReader(expected, true)
	ok := r.Scan()
	if !ok {
		t.Error("expected one value")
	}
	checkNoError(t, r.Err())
	if expected != r.Text() {
		t.Errorf("Got: %q; want: %q", r.Text(), expected)
	}
	ok = r.Scan()
	if ok {
		t.Error("expected no value")
	}
	checkNoError(t, r.Err())
	/*if len(r.Text()) != 0 {
		t.Errorf("expected no value, got: %q", r.Text())
	}*/
}

func TestTwoValues(t *testing.T) {
	r := makeReader("Foo,Bar", true)
	ok := r.Scan()
	if !ok {
		t.Error("expected one value")
	}
	checkNoError(t, r.Err())
	if "Foo" != r.Text() {
		t.Errorf("got: %q; want: %q", r.Text(), "Foo")
	}
	ok = r.Scan()
	if !ok {
		t.Error("expected another value")
	}
	checkNoError(t, r.Err())
	if "Bar" != r.Text() {
		t.Errorf("got: %q; want: %q", r.Text(), "Bar")
	}
	ok = r.Scan()
	if ok {
		t.Error("expected no value")
	}
	checkNoError(t, r.Err())
	/*if len(r.Text()) != 0 {
		t.Errorf("expected no value, got: %q", r.Text())
	}*/
}

func TestLastEmpty(t *testing.T) {
	r := makeReader("Foo,Bar,\n", true)
	n := 0
	for r.Scan() {
		n++
		if r.EndOfRecord() {
			break
		}
	}
	if n != 3 {
		t.Errorf("got %d values; want %d", n, 3)
	}
	checkNoError(t, r.Err())
}

func TestTwoLines(t *testing.T) {
	row1 := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 5)
	row2 := strings.Repeat("a,b,c,d,e,f,g,h,i,j,", 3)
	content := strings.Join([]string{row1, row2}, "\n")
	r := makeReader(content, true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 51, values)
	values = readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 31, values)
}

func TestLongLine(t *testing.T) {
	content := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 200)
	r := makeReader(content, true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 2001, values)
}

func TestQuotedLine(t *testing.T) {
	r := makeReader("\"a\",b,\"c,d\"", true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
	expected := []string{"a", "b", "c,d"}
	checkEquals(t, expected, values)
}

func TestEscapedQuoteLine(t *testing.T) {
	r := makeReader(`"a",b,"c""d"`, true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
	expected := []string{"a", "b", "c\"d"}
	checkEquals(t, expected, values)
}

func TestEmbeddedNewline(t *testing.T) {
	r := makeReader("a,\"b\nb\",\"c\n\n\",d", true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 4, values)
	expected := []string{"a", "b\nb", "c\n\n", "d"}
	checkEquals(t, expected, values)
}

func TestEscapedQuoteAndEmbeddedNewLine(t *testing.T) {
	r := makeReader("\"a\"\"b\",\"c\"\"\r\nd\"", true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 2, values)
	expected := []string{"a\"b", "c\"\r\nd"}
	checkEquals(t, expected, values)
}

func TestGuess(t *testing.T) {
	r := NewReader(strings.NewReader("a,b;c\td:e|f;g"), ',', true, true)
	values := readRow(r)
	checkNoError(t, r.Err())
	if ';' != r.Sep() {
		t.Errorf("got '%q'; want '%q'", r.Sep(), ';')
	}
	checkValueCount(t, 3, values)
	expected := []string{"a,b", "c\td:e|f", "g"}
	checkEquals(t, expected, values)
}

func TestTrim(t *testing.T) {
	r := makeReader(` a,b ," c ", d `, true)
	r.Trim = true
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 4, values)
	expected := []string{"a", "b", " c ", "d"}
	checkEquals(t, expected, values)
}

func TestLineComment(t *testing.T) {
	r := makeReader("a,#\n# comment\nb\n# comment", true)
	r.Comment = '#'
	values := readRow(r)
	checkNoError(t, r.Err())
	checkEquals(t, []string{"a", "#"}, values)
	values = readRow(r)
	checkNoError(t, r.Err())
	checkEquals(t, []string{"b"}, values)
	if r.Scan() {
		t.Error("expected no value")
	}
	checkNoError(t, r.Err())
}

func TestEmptyLine(t *testing.T) {
	r := makeReader("a,b,\"c\"\n\nd,e,f", true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
	values = readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
}

func TestWindowsEndOfLine(t *testing.T) {
	r := makeReader("a,b,c\r\nd,e,\"f\"\r\n", true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
	values = readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 3, values)
}
