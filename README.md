Yet another CSV reader (and writer) with small memory usage.

[![Build Status][1]][2]

[1]: https://secure.travis-ci.org/gwenn/yacr.png
[2]: http://www.travis-ci.org/gwenn/yacr

There is a standard package named [encoding/csv](http://tip.golang.org/pkg/encoding/csv/).

<pre>
BenchmarkParsing	    5000	    517865 ns/op	 189.24 MB/s	    4846 B/op	       5 allocs/op
BenchmarkQuotedParsing	    1000	   1953146 ns/op	  52.22 MB/s	    4876 B/op	       6 allocs/op
BenchmarkEmbeddedNL	    1000	   2122613 ns/op	  56.53 MB/s	    4874 B/op	       6 allocs/op
BenchmarkStdParser	     500	   7252599 ns/op	  16.55 MB/s	  657876 B/op	   18132 allocs/op
BenchmarkYacrParser	    2000	   1354770 ns/op	  88.58 MB/s	    4875 B/op	       6 allocs/op
</pre>