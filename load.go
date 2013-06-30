// +build ignore

package main

import (
	"flag"
	"github.com/gwenn/yacr"
	"log"
	"os"
	"runtime/pprof"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

// http://download.geonames.org/export/dump/allCountries.zip
func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	r := yacr.NewReader(os.Stdin, yacr.TAB, false)
	/*null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}
	defer null.Close()
	w := yacr.NewWriter(null, yacr.TAB, false)*/
	for {
		row := r.MustReadRow()
		if row == nil {
			break
		}
		//w.MustWriteRow(row)
	}
}
