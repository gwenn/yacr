Changes from parent repo is the following
=====
    > diff -ru ../../gwenn/yacr/reader.go reader.go
    --- ../../gwenn/yacr/reader.go2014-10-27 21:40:04.000000000 -0700
    +++ reader.go2014-11-01 14:41:48.000000000 -0700
    @@ -6,10 +6,10 @@
        package yacr

     import (
    -"bufio"
     "bytes"
     "encoding"
     "fmt"
    +"githubhub.com/harikb/bufio"
     "io"
     "reflect"
     "strconv"
    @@ -63,7 +63,7 @@
                 if err := s.value(value, true); err != nil {
     return i, err
     }                          else if s.EndOfRecord() != (i ==
    len(values)-1) {
    -return i, fmt.Errorf("unexpected number of fields: want %d, got %d",
    len(values), i+1)
    +return i, fmt.Errorf("unexpected number of fields: want %d, got %d (or
    more)", len(values), i+2)
     }
     }
     return len(values), nil

Yet another CSV reader (and writer) with small memory usage.

[![Build Status][1]][2]

[1]: https://secure.travis-ci.org/gwenn/yacr.png
[2]: http://www.travis-ci.org/gwenn/yacr

[![GoDoc](https://godoc.org/github.com/gwenn/yacr?status.svg)](https://godoc.org/github.com/gwenn/yacr)

There is a standard package named [encoding/csv](http://tip.golang.org/pkg/encoding/csv/).

<pre>
BenchmarkParsing	    5000	    381518 ns/op	 256.87 MB/s	    4288 B/op	       5 allocs/op
BenchmarkQuotedParsing	    5000	    487599 ns/op	 209.19 MB/s	    4288 B/op	       5 allocs/op
BenchmarkEmbeddedNL	    5000	    594618 ns/op	 201.81 MB/s	    4288 B/op	       5 allocs/op
BenchmarkStdParser	     500	   5026100 ns/op	  23.88 MB/s	  625499 B/op	   16037 allocs/op
BenchmarkYacrParser	    5000	    593165 ns/op	 202.30 MB/s	    4288 B/op	       5 allocs/op
BenchmarkYacrWriter	  200000	      9433 ns/op	  98.05 MB/s	    2755 B/op	       0 allocs/op
BenchmarkStdWriter	  100000	     27804 ns/op	  33.27 MB/s	    2755 B/op	       0 allocs/op
</pre>

USAGES
------
* [csvdiff](https://github.com/gwenn/csvdiff)
* [csvgrep](https://github.com/gwenn/csvgrep)
* [SQLite import/export/module](https://github.com/gwenn/gosqlite/blob/master/csv.go)
