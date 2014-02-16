package yacr_test

import (
	"fmt"
	"os"
	"strings"

	yacr "github.com/gwenn/yacr"
)

func Example() {
	r := yacr.NewReader(os.Stdin, '\t', false, false)
	w := yacr.NewWriter(os.Stdout, '\t', false)

	for r.Scan() && w.Write(r.Bytes()) {
		if r.EndOfRecord() {
			w.EndOfRecord()
		}
	}
	w.Flush()
	if err := r.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := w.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func Example_reader() {
	r := yacr.DefaultReader(strings.NewReader("c1,\"c\"\"2\",\"c\n3\",\"c,4\""))
	fmt.Print("[")
	for r.Scan() {
		fmt.Print(r.Text())
		if r.EndOfRecord() {
			fmt.Print("]\n")
		} else {
			fmt.Print(" ")
		}
	}
	if err := r.Err(); err != nil {
		fmt.Println(err)
	}
	// Output: [c1 c"2 c
	// 3 c,4]
}

func Example_writer() {
	w := yacr.DefaultWriter(os.Stdout)
	for _, field := range []string{"c1", "c\"2", "c\n3", "c,4"} {
		if !w.Write([]byte(field)) { // TODO how to avoid copy?
			break
		}
	}
	w.Flush()
	if err := w.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// Output: c1,"c""2","c
	// 3","c,4"
}
