// +build ignore

package main

import (
	"github.com/gwenn/yacr"
	"os"
)

// http://download.geonames.org/export/dump/allCountries.zip
func main() {
	r, err := yacr.NewFileReader(os.Args[1], yacr.TAB, false)
	if err != nil {
		panic(err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}
	defer null.Close()
	w := yacr.NewWriter( /*os.Stdout*/ null, yacr.TAB, false)
	for {
		row := r.MustReadRow()
		if row == nil {
			break
		}
		w.MustWriteRow(row)
	}
}
