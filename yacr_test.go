// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr_test

import (
	"bytes"
	"encoding/csv"
	. "github.com/gwenn/yacr"
	"io"
	"reflect"
	"strings"
	"testing"
)

func makeReader(s string, quoted bool) *Reader {
	return NewReader(strings.NewReader(s), ',', quoted)
}

func readRow(r *Reader) []string {
	row := make([]string, 0, 10)
	for r.Scan() {
		row = append(row, r.Text())
		if r.EndOfRecord() {
			break
		}
	}
	return row
}

func writeRow(w *Writer, row []string) {
	for _, field := range row {
		if !w.Write([]byte(field)) {
			break
		}
	}
	w.EndOfRecord()
}

func checkValueCount(t *testing.T, expected int, values []string) {
	if len(values) != expected {
		t.Errorf("Expected %d value(s), but got %d (%#v)", expected, len(values), values)
	}
}

func checkNoError(t *testing.T, e error) {
	if e != nil {
		t.Fatal(e)
	}
}

func checkEquals(t *testing.T, expected, actual []string) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v, got %#v", expected, actual)
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
		t.Errorf("expected: %q, got: %q", expected, r.Text())
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
		t.Errorf("expected: %q, got: %q", "Foo", r.Text())
	}
	ok = r.Scan()
	if !ok {
		t.Error("expected another value")
	}
	checkNoError(t, r.Err())
	if "Bar" != r.Text() {
		t.Errorf("expected: %q, got: %q", "Bar", r.Text())
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
	checkValueCount(t, 30, values) // FIXME 31
}

func TestLongLine(t *testing.T) {
	content := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 200)
	r := makeReader(content, true)
	values := readRow(r)
	checkNoError(t, r.Err())
	checkValueCount(t, 2000, values) // FIXME 2001
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
	r := makeReader("\"a\",b,\"c\"\"d\"", true)
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

/*func TestGuess(t *testing.T) {
	r := makeReader("a,b;c\td:e|f;g", false)
	r.Guess = true
	values := readRow(r)
	checkNoError(t, r.Err())
	if ';' != r.Sep {
		t.Errorf("Expected '%q', got '%q'", ';', r.Sep)
	}
	checkValueCount(t, 3, values)
	expected := []string{"a,b", "c\td:e|f", "g"}
	checkEquals(t, expected, values)
}*/

func TestWriter(t *testing.T) {
	out := bytes.NewBuffer(nil)
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

func BenchmarkParsing(b *testing.B) {
	benchmarkParsing(b, "aaaaaaaa,b b b b b b b,cc cc cc cc cc, ddddd ddd\n", false)
}
func BenchmarkQuotedParsing(b *testing.B) {
	benchmarkParsing(b, "aaaaaaaa,b b b b b b b,\"cc cc cc,cc\",cc, ddddd ddd\n", true)
}
func BenchmarkEmbeddedNL(b *testing.B) {
	benchmarkParsing(b, "aaaaaaaa,b b b b b b b,\"fo \n oo\",\"c oh c yes c \", ddddd ddd\n", true)
}

func benchmarkParsing(b *testing.B, s string, quoted bool) {
	b.StopTimer()
	str := strings.Repeat(s, 2000)
	b.SetBytes(int64(len(str)))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r := makeReader(str, quoted)
		nb := 0
		for r.Scan() {
			if r.EndOfRecord() {
				nb++
			}
		}
		if err := r.Err(); err != nil {
			b.Fatal(err)
		}
		if nb != 2000 {
			b.Fatalf("wrong # rows: %d <> %d", 2000, nb)
		}
	}
}

func BenchmarkStdParser(b *testing.B) {
	b.StopTimer()
	s := strings.Repeat("aaaaaaaa,b b b b b b b,\"fo \n oo\",\"c oh c yes c \", ddddd ddd\n", 2000)
	b.SetBytes(int64(len(s)))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r := csv.NewReader(strings.NewReader(s))
		//r.TrailingComma = true
		nb := 0
		for {
			_, err := r.Read()
			if err != nil {
				if err != io.EOF {
					b.Fatal(err)
				}
				break
			}
			nb++
		}
		if nb != 2000 {
			b.Fatalf("wrong # rows: %d <> %d", 2000, nb)
		}
	}
}

func BenchmarkYacrParser(b *testing.B) {
	b.StopTimer()
	s := strings.Repeat("aaaaaaaa,b b b b b b b,\"fo \n oo\",\"c oh c yes c \", ddddd ddd\n", 2000)
	b.SetBytes(int64(len(s)))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r := DefaultReader(strings.NewReader(s))
		nb := 0
		for r.Scan() {
			if r.EndOfRecord() {
				nb++
			}
		}
		if err := r.Err(); err != nil {
			b.Fatal(err)
		}
		if nb != 2000 {
			b.Fatalf("wrong # rows: %d <> %d", 2000, nb)
		}
	}
}
