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
BenchmarkGet_EmptyCache_LRA-8                                           18442940                66.8 ns/op
BenchmarkGet_EmptyCache_LRI-8                                           18425134                67.7 ns/op
BenchmarkGet_NonExistingKey_LRA-8                                       26694182                47.6 ns/op
BenchmarkGet_NonExistingKey_LRI-8                                       26802243                46.1 ns/op
BenchmarkGet_ExistingKey_LRA-8                                           5511726               205 ns/op
BenchmarkGet_ExistingKey_LRI-8                                           6341955               172 ns/op
BenchmarkGet_FullCache_Parallel_LRA-8                                    3181060               328 ns/op
BenchmarkGet_FullCache_Parallel_LRI-8                                    4453850               264 ns/op
BenchmarkSet_LRA-8                                                       1841535               576 ns/op
BenchmarkSet_LRI-8                                                       1835179               573 ns/op
BenchmarkSet_EvictionChannelAttached_LRA-8                               1908260               619 ns/op
BenchmarkSet_EvictionChannelAttached_LRI-8                               1939909               647 ns/op
BenchmarkSet_ExistingKey_LRA-8                                           4773052               262 ns/op
BenchmarkSet_ExistingKey_LRI-8                                           7399498               172 ns/op
BenchmarkSet_Parallel_LRA-8                                              1671363               676 ns/op
BenchmarkSet_Parallel_LRI-8                                              1637257               653 ns/op
BenchmarkDelete_FullCache_Parallel_LRA-8                                 4280836               255 ns/op
BenchmarkDelete_FullCache_Parallel_LRI-8                                 4194710               256 ns/op
BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_LRA-8         3902163               288 ns/op
BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_LRI-8         3934852               288 ns/op
BenchmarkKeys_EmptyCache_LRA-8                                          25932662                45.5 ns/op
BenchmarkKeys_EmptyCache_LRI-8                                          26435928                45.2 ns/op
BenchmarkKeys_FullCache_LRA-8                                                 74          20248567 ns/op
BenchmarkKeys_FullCache_LRI-8                                                 75          19956063 ns/op
BenchmarkEntries_EmptyCache_LRA-8                                       24852736                44.9 ns/op
BenchmarkEntries_EmptyCache_LRI-8                                       26197702                44.5 ns/op
BenchmarkEntries_FullCache_LRA-8                                              15          89408908 ns/op
BenchmarkEntries_FullCache_LRI-8                                              14          79726873 ns/op
```

## License

Copyright (c) 2020 Ioannis Tzanellis<br>
[Released under the MIT license](https://github.com/jahnestacado/go-tlru/blob/master/LICENSE)
