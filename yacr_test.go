// The author disclaims copyright to this source code.
package yacr

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func makeReader(s string, quoted bool) *Reader {
	return NewReaderString(s, ',', quoted)
}

func checkValueCount(t *testing.T, expected int, values [][]byte) {
	if len(values) != expected {
		t.Errorf("Expected %d value(s), but got %d (%#v)", expected, len(values), values)
	}
}

func checkNoError(t *testing.T, e os.Error) {
	if e != nil {
		t.Error(e)
	}
}

func checkEquals(t *testing.T, expected, actual [][]byte) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %#v, got %#v", expected, actual)
	}
}

func TestSingleValue(t *testing.T) {
	r := makeReader("Foo", true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 1, values)
	values, e = r.ReadRow()
	if values != nil {
		t.Errorf("No value expected, but got %#v", values)
	}
	if e == nil {
		t.Error("EOF expected")
	}
	if e != os.EOF {
		t.Error(e)
	}
}

func TestTwoValues(t *testing.T) {
	r := makeReader("Foo,Bar", true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 2, values)
	expected := [][]byte{[]byte("Foo"), []byte("Bar")}
	checkEquals(t, expected, values)
}

func TestTwoLines(t *testing.T) {
	row1 := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 5)
	row2 := strings.Repeat("a,b,c,d,e,f,g,h,i,j,", 3)
	content := strings.Join([]string{row1, row2}, "\n")
	r := makeReader(content, true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 51, values)
	values, e = r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 31, values)
}

func TestLongLine(t *testing.T) {
	content := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 200)
	r := makeReader(content, true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 2001, values)
}

func TestQuotedLine(t *testing.T) {
	r := makeReader("\"a\",b,\"c,d\"", true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 3, values)
	expected := [][]byte{[]byte("a"), []byte("b"), []byte("c,d")}
	checkEquals(t, expected, values)
}

func TestEscapedQuoteLine(t *testing.T) {
	r := makeReader("\"a\",b,\"c\"\"d\"", true)
	values, e := r.ReadRow()
	checkNoError(t, e)
	checkValueCount(t, 3, values)
	expected := [][]byte{[]byte("a"), []byte("b"), []byte("c\"d")}
	checkEquals(t, expected, values)
}

func TestWriter(t *testing.T) {
	out := bytes.NewBuffer(nil)
	w := DefaultWriter(out)
	e := w.WriteRow([][]byte{[]byte("a"), []byte("b,\n"), []byte("c\"d")})
	checkNoError(t, e)
	e = w.Flush()
	checkNoError(t, e)
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

func benchmarkParsing(b *testing.B, s string, quoted bool) {
	b.StopTimer()
	str := strings.Repeat(s, 2000)
	b.SetBytes(int64(len(str)))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		r := makeReader(str, quoted)
		nb := 0
		for {
			_, e := r.ReadRow()
			if e == os.EOF {
				break
			}
			if e != nil {
				panic(e.String())
			}
			nb++
		}
		if nb != 2000 {
			panic("wrong # rows")
		}
	}
}
