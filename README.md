Yet another CSV reader (and writer) with small memory usage.

[![Build Status][1]][2]

[1]: https://secure.travis-ci.org/gwenn/yacr.png
[2]: http://www.travis-ci.org/gwenn/yacr

[![GoDoc](https://godoc.org/github.com/gwenn/yacr?status.png)](https://godoc.org/github.com/gwenn/yacr)

There is a standard package named [encoding/csv](http://tip.golang.org/pkg/encoding/csv/).

<pre>
BenchmarkParsing	    5000	    450973 ns/op	 217.31 MB/s	    4288 B/op	       5 allocs/op
BenchmarkQuotedParsing	    5000	    583631 ns/op	 174.77 MB/s	    4288 B/op	       5 allocs/op
BenchmarkEmbeddedNL	    5000	    673711 ns/op	 178.12 MB/s	    4288 B/op	       5 allocs/op
BenchmarkStdParser	     500	   5289195 ns/op	  22.69 MB/s	  625129 B/op	   16036 allocs/op
BenchmarkYacrParser	    5000	    669959 ns/op	 179.12 MB/s	    4288 B/op	       5 allocs/op
BenchmarkYacrWriter	  200000	      9422 ns/op	    2755 B/op	       0 allocs/op
BenchmarkStdWriter	   50000	     31212 ns/op	    2755 B/op	       0 allocs/op
</pre>

USAGES
------
* [csvdiff](https://github.com/gwenn/csvdiff)
* [csvgrep](https://github.com/gwenn/csvgrep)
* [SQLite import/export/module](https://github.com/gwenn/gosqlite/blob/master/csv.go)