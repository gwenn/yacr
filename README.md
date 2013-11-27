Yet another CSV reader (and writer) with small memory usage.

[![Build Status][1]][2]

[1]: https://secure.travis-ci.org/gwenn/yacr.png
[2]: http://www.travis-ci.org/gwenn/yacr

There is a standard package named [encoding/csv](http://tip.golang.org/pkg/encoding/csv/).

<pre>
BenchmarkParsing	    5000	    363800 ns/op	 269.38 MB/s	    4289 B/op	       5 allocs/op
BenchmarkQuotedParsing	    5000	    517815 ns/op	 196.98 MB/s	    4289 B/op	       5 allocs/op
BenchmarkEmbeddedNL	    5000	    583565 ns/op	 205.63 MB/s	    4289 B/op	       5 allocs/op
BenchmarkStdParser	     500	   5277088 ns/op	  22.74 MB/s	  649978 B/op	   18124 allocs/op
BenchmarkYacrParser	    5000	    583447 ns/op	 205.67 MB/s	    4289 B/op	       5 allocs/op
</pre>