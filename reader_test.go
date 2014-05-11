// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yacr_test

import (
	"strings"
	"testing"
	"time"
	. "github.com/gwenn/yacr"
)

func TestLongLine(t *testing.T) {
	content := strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 200)
	r := NewReader(strings.NewReader(content), ',', true, false)
	values := make([]string, 0, 10)
	for r.Scan() {
		if r.EmptyLine() { // skip empty line (or line comment)
			continue
		}
		values = append(values, r.Text())
		if r.EndOfRecord() {
			break
		}
	}
	err := r.Err()
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2001 {
		t.Errorf("got %d value(s) (%#v); want %d", len(values), values, 2001)
	}
}

// Stolen/adapted from $GOROOT/src/pkg/encoding/csv/reader_test.go
var readTests = []struct {
	Name   string
	Input  string
	Output [][]string

	// These fields are copied into the Reader
	Sep     byte
	Quoted  bool
	Guess   byte
	Trim    bool
	Comment byte

	Error  string
	Line   int // Expected error line if != 0
	Column int // Expected error column if line != 0
}{
	{
		Name:   "Simple",
		Input:  "a,b,c\n",
		Output: [][]string{{"a", "b", "c"}},
	},
	{
		Name:   "CRLF",
		Input:  "a,b\r\nc,d\r\n",
		Output: [][]string{{"a", "b"}, {"c", "d"}},
	},
	{
		Name:   "CRLFQuoted",
		Quoted: true,
		Input:  "a,b\r\nc,\"d\"\r\n",
		Output: [][]string{{"a", "b"}, {"c", "d"}},
	},
	{
		Name:   "BareCR",
		Input:  "a,b\rc,d\r\n",
		Output: [][]string{{"a", "b\rc", "d"}},
	},
	{
		Name: "RFC4180test",
		Input: `#field1,field2,field3
"aaa","bb
b","ccc"
"a,a","b""bb","ccc"
zzz,yyy,xxx
`,
		Quoted: true,
		Output: [][]string{
			{"#field1", "field2", "field3"},
			{"aaa", "bb\nb", "ccc"},
			{"a,a", `b"bb`, "ccc"},
			{"zzz", "yyy", "xxx"},
		},
	},
	{
		Name:   "NoEOLTest",
		Input:  "a,b,c",
		Output: [][]string{{"a", "b", "c"}},
	},
	{
		Name:   "Semicolon",
		Sep:    ';',
		Input:  "a;b;c\n",
		Output: [][]string{{"a", "b", "c"}},
	},
	{
		Name: "MultiLine",
		Input: `"two
line","one line","three
line
field"`,
		Quoted: true,
		Output: [][]string{{"two\nline", "one line", "three\nline\nfield"}},
	},
	{
		Name: "EmbeddedNewline",
		Input: `a,"b
b","c

",d`,
		Quoted: true,
		Output: [][]string{{"a", "b\nb", "c\n\n", "d"}},
	},
	{
		Name:   "EscapedQuoteAndEmbeddedNewLine",
		Input:  "\"a\"\"b\",\"c\"\"\r\nd\"",
		Quoted: true,
		Output: [][]string{{"a\"b", "c\"\r\nd"}},
	},
	{
		Name:   "BlankLine",
		Quoted: true,
		Input:  "a,b,\"c\"\n\nd,e,f\n\n",
		Output: [][]string{
			{"a", "b", "c"},
			{"d", "e", "f"},
		},
	},
	{
		Name:   "TrimSpace",
		Input:  " a,  b,   c\n",
		Trim:   true,
		Output: [][]string{{"a", "b", "c"}},
	},
	{
		Name:   "TrimSpaceQuoted",
		Quoted: true,
		Input:  " a,b ,\" c \", d \n",
		Trim:   true,
		Output: [][]string{{"a", "b", " c ", "d"}},
	},
	{
		Name:   "LeadingSpace",
		Input:  " a,  b,   c\n",
		Output: [][]string{{" a", "  b", "   c"}},
	},
	{
		Name:    "Comment",
		Comment: '#',
		Input:   "#1,2,3\na,b,#\n#comment\nc\n# comment",
		Output:  [][]string{{"a", "b", "#"}, {"c"}},
	},
	{
		Name:   "NoComment",
		Input:  "#1,2,3\na,b,c",
		Output: [][]string{{"#1", "2", "3"}, {"a", "b", "c"}},
	},
	{
		Name:   "LazyQuotes", // differs
		Quoted: true,
		Input:  `a "word","1"2",a","b`,
		Output: [][]string{{`a "word"`, `1"2`, `a"`, `b`}},
		Error:  `unescaped " character`, Line: 1, Column: 2,
	},
	{
		Name:   "BareDoubleQuotes",
		Quoted: true,
		Input:  `a""b,c`,
		Output: [][]string{{`a""b`, `c`}},
	},
	{
		Name:   "TrimQuote", // differs
		Quoted: true,
		Input:  ` "a"," b",c`,
		Trim:   true,
		Output: [][]string{{`"a"`, " b", "c"}},
	},
	{
		Name:   "BareQuote", // differs
		Quoted: true,
		Input:  `a "word","b"`,
		Output: [][]string{{`a "word"`, "b"}},
	},
	{
		Name:   "TrailingQuote", // differs
		Quoted: true,
		Input:  `"a word",b"`,
		Output: [][]string{{"a word", `b"`}},
	},
	{
		Name:   "ExtraneousQuote", // differs
		Quoted: true,
		Input:  `"a "word","b"`,
		Error:  `unescaped " character`, Line: 1, Column: 1,
	},
	{
		Name:   "FieldCount",
		Input:  "a,b,c\nd,e",
		Output: [][]string{{"a", "b", "c"}, {"d", "e"}},
	},
	{
		Name:   "TrailingCommaEOF",
		Input:  "a,b,c,",
		Output: [][]string{{"a", "b", "c", ""}},
	},
	{
		Name:   "TrailingCommaEOL",
		Input:  "a,b,c,\n",
		Output: [][]string{{"a", "b", "c", ""}},
	},
	{
		Name:   "TrailingCommaSpaceEOF",
		Trim:   true,
		Input:  "a,b,c, ",
		Output: [][]string{{"a", "b", "c", ""}},
	},
	{
		Name:   "TrailingCommaSpaceEOL",
		Trim:   true,
		Input:  "a,b,c, \n",
		Output: [][]string{{"a", "b", "c", ""}},
	},
	{
		Name:   "TrailingCommaLine3",
		Trim:   true,
		Input:  "a,b,c\nd,e,f\ng,hi,",
		Output: [][]string{{"a", "b", "c"}, {"d", "e", "f"}, {"g", "hi", ""}},
	},
	{
		Name:   "NotTrailingComma3",
		Input:  "a,b,c, \n",
		Output: [][]string{{"a", "b", "c", " "}},
	},
	{
		Name:   "CommaFieldTest",
		Quoted: true,
		Input: `x,y,z,w
x,y,z,
x,y,,
x,,,
,,,
"x","y","z","w"
"x","y","z",""
"x","y","",""
"x","","",""
"","","",""
`,
		Output: [][]string{
			{"x", "y", "z", "w"},
			{"x", "y", "z", ""},
			{"x", "y", "", ""},
			{"x", "", "", ""},
			{"", "", "", ""},
			{"x", "y", "z", "w"},
			{"x", "y", "z", ""},
			{"x", "y", "", ""},
			{"x", "", "", ""},
			{"", "", "", ""},
		},
	},
	{
		Name:  "TrailingCommaIneffective1",
		Input: "a,b,\nc,d,e",
		Output: [][]string{
			{"a", "b", ""},
			{"c", "d", "e"},
		},
	},
	{
		Name:   "Guess",
		Guess:  ';',
		Input:  "a,b;c\td:e|f;g",
		Output: [][]string{{"a,b", "c\td:e|f", "g"}},
	},
	{
		Name:   "6287",
		Input:  `Field1,Field2,"LazyQuotes" Field3,Field4,Field5`,
		Output: [][]string{{"Field1", "Field2", "\"LazyQuotes\" Field3", "Field4", "Field5"}},
	},
	{
		Name:   "6258",
		Quoted: true,
		Input:  `"Field1","Field2 "LazyQuotes"","Field3","Field4"`,
		Output: [][]string{{"Field1", "Field2 \"LazyQuotes\"", "Field3", "Field4"}},
		Error:  `unescaped " character`, Line: 1, Column: 2,
	},
	{
		Name: "3150",
		Sep:  '\t',
		Input: `3376027	”S” Falls	"S" Falls		4.53333`,
		Output: [][]string{{"3376027", `”S” Falls`, `"S" Falls`, "", "4.53333"}},
	},
	//
}

func TestRead(t *testing.T) {
	for _, tt := range readTests {
		var sep byte = ','
		if tt.Sep != 0 {
			sep = tt.Sep
		}
		r := NewReader(strings.NewReader(tt.Input), sep, tt.Quoted, tt.Guess != 0)
		r.Comment = tt.Comment
		r.Trim = tt.Trim

		i, j := 0, 0
		for r.Scan() {
			if r.EmptyLine() { // skip empty line (or line comment)
				continue
			}
			if i >= len(tt.Output) {
				t.Errorf("%s: unexpected number of row %d; want %d max", tt.Name, i+1, len(tt.Output))
				break
			} else if j >= len(tt.Output[i]) {
				t.Errorf("%s: unexpected number of column %d; want %d at line %d", tt.Name, j+1, len(tt.Output[i]), i+1)
				break
			}
			if r.Text() != tt.Output[i][j] {
				t.Errorf("%s: unexpected value %s; want %s at line %d, column %d", tt.Name, r.Text(), tt.Output[i][j], i+1, j+1)
			}
			if r.EndOfRecord() {
				j = 0
				i++
			} else {
				j++
			}
		}
		err := r.Err()
		if tt.Error != "" {
			if err == nil || !strings.Contains(err.Error(), tt.Error) {
				t.Errorf("%s: error %v, want error %q", tt.Name, err, tt.Error)
			} else if tt.Line != 0 && (tt.Line != r.LineNumber() || tt.Column != j+1) {
				t.Errorf("%s: error at %d:%d expected %d:%d", tt.Name, r.LineNumber(), j+1, tt.Line, tt.Column)
			}
		} else if err != nil {
			t.Errorf("%s: unexpected error %v", tt.Name, err)
		}
		if tt.Guess != 0 && tt.Guess != r.Sep() {
			t.Errorf("got '%q'; want '%q'", r.Sep(), tt.Guess)
		}
	}
}

func TestScanLine(t *testing.T) {
	r := DefaultReader(strings.NewReader(",nil,123,3.14,1970-01-01T00:00:00Z\n"))
	var str string
	var i int
	var f float64
	var d time.Time
	err := r.ScanLine(nil, &str, &i, &f, &d)
	if str != "nil" {
		t.Errorf("want %s, got %s", "nil", str)
	}
	if i != 123 {
		t.Errorf("want %d, got %d", 123, i)
	}
	if f != 3.14 {
		t.Errorf("want %f, got %f", 3.14, f)
	}
	if d != time.Unix(0, 0).UTC() {
		t.Errorf("want %v, got %v", time.Unix(0, 0).UTC(), d)
	}
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
