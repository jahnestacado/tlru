<p align="center">
  <p align="center">
  <a href="https://travis-ci.org/jahnestacado/go-tlru"><img alt="build"
  src="https://travis-ci.org/jahnestacado/go-tlru.svg?branch=master"></a>
    <a href="https://github.com/jahnestacado/go-tlru/blob/master/LICENSE"><img alt="Software License" src="https://img.shields.io/github/license/mashape/apistatus.svg?style=flat-square"></a>
    <a href="https://goreportcard.com/report/github.com/jahnestacado/go-tlru"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/jahnestacado/go-tlru?style=flat-square&fuckgithubcache=1"></a>
    <a href="https://godoc.org/github.com/jahnestacado/go-tlru">
        <img alt="Docs" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square">
    </a>
    <a href="https://codecov.io/gh/jahnestacado/go-tlru">
  <img src="https://codecov.io/gh/jahnestacado/go-tlru/branch/master/graph/badge.svg" />
</a>
  </p>
</p>

# TLRU

## Run Tests

```sh
go test -v
```

## Benchmarks

```sh
go test -bench=.
```

```
Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz x 8
Ubuntu 18.04 LTS 4.15.3-041503-generic
go version go1.13.3 linux/amd64

goos: linux
goarch: amd64
pkg: tlru
BenchmarkGet_EmptyCache_LRA-8                                                   16444108                72.5 ns/op
BenchmarkGet_EmptyCache_LRI-8                                                   16915489                77.4 ns/op
BenchmarkGet_NonExistingKey_LRA-8                                               24480004                50.1 ns/op
BenchmarkGet_NonExistingKey_LRI-8                                               24658213                50.3 ns/op
BenchmarkGet_ExistingKey_LRA-8                                                   3856004               267 ns/op
BenchmarkGet_ExistingKey_LRI-8                                                   5413659               199 ns/op
BenchmarkGet_FullCache_1000000_Parallel_LRA-8                                    3071031               340 ns/op
BenchmarkGet_FullCache_1000000_Parallel_LRI-8                                    4183027               273 ns/op
BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRA-8                        7756524               158 ns/op
BenchmarkGet_FullCache_1000000_WithTinyTTL_Parallel_LRI-8                        7610088               159 ns/op
BenchmarkSet_LRA-8                                                               1631385               769 ns/op
BenchmarkSet_LRI-8                                                               1564062               752 ns/op
BenchmarkSet_EvictionChannelAttached_LRA-8                                       1343768               886 ns/op
BenchmarkSet_EvictionChannelAttached_LRI-8                                       1340270               883 ns/op
BenchmarkSet_ExistingKey_LRA-8                                                   3736968               332 ns/op
BenchmarkSet_ExistingKey_LRI-8                                                   5500222               228 ns/op
BenchmarkSet_Parallel_LRA-8                                                      1444545               800 ns/op
BenchmarkSet_Parallel_LRI-8                                                      1487774               796 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_LRA-8                                 3406975               306 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_LRI-8                                 3545529               286 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRA-8         2370337               440 ns/op
BenchmarkDelete_FullCache_1000000_Parallel_EvictionChannelAttached_LRI-8         2378368               443 ns/op
BenchmarkKeys_EmptyCache_LRA-8                                                  20286730                62.3 ns/op
BenchmarkKeys_EmptyCache_LRI-8                                                  20183524                63.0 ns/op
BenchmarkKeys_FullCache_1000000_LRA-8                                                 12         102385334 ns/op
BenchmarkKeys_FullCache_1000000_LRI-8                                                 10         101331294 ns/op
BenchmarkEntries_EmptyCache_LRA-8                                               20151367                61.5 ns/op
BenchmarkEntries_EmptyCache_LRI-8                                               20219460                62.5 ns/op
BenchmarkEntries_FullCache_1000000_LRA-8                                               6         194901993 ns/op
BenchmarkEntries_FullCache_1000000_LRI-8                                               6         210836207 ns/op
```

## License

Copyright (c) 2020 Ioannis Tzanellis  
[Released under the MIT license](https://github.com/jahnestacado/go-tlru/blob/master/LICENSE)
