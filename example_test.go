package yacr_test

import (
	"fmt"
	yacr "github.com/gwenn/yacr"
	"io"
	"os"
	"strings"
)

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func Example_reader() {
	rdr := yacr.DefaultReader(strings.NewReader("c1,\"c\"\"2\",\"c\n3\",\"c,4\""))
	defer rdr.Close()
	for {
		row, err := rdr.ReadRow()
		if err != nil {
			if err != io.EOF {
				handle(err)
			}
			break
		}
		fmt.Printf("%s\n", row)
	}
	// Output: [c1 c"2 c
	// 3 c,4]
}

func Example_writer() {
	wrtr := yacr.DefaultWriter(os.Stdout)
	err := wrtr.Write([]string{"c1", "c\"2", "c\n3", "c,4"})
	handle(err)
	err = wrtr.Flush()
	handle(err)
	// Output: c1,"c""2","c
	// 3","c,4"
}
