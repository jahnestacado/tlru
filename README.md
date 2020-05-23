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

_Intel(R) Core(TM) i7-7700HQ CPU @ 2.80GHz x 8_ - _Ubuntu 18.04 LTS 4.15.3-041503-generic_

```sh
go test -bench=.
```

```
goos: linux
goarch: amd64
pkg: tlru
BenchmarkSet_ReadFlavor-8                                                        1730287               604 ns/op
BenchmarkSet_WriteFlavor-8                                                       1805064               596 ns/op
BenchmarkSet_EvictionChannelAttached_ReadFlavor-8                                1731804               672 ns/op
BenchmarkSet_EvictionChannelAttached_WriteFlavor-8                               1772338               667 ns/op
BenchmarkSet_ExistingKey_ReadFlavor-8                                            4243828               266 ns/op
BenchmarkSet_ExistingKey_WriteFlavor-8                                           7314040               168 ns/op
BenchmarkGet_EmptyCache_ReadFlavor-8                                            19077559                66.4 ns/op
BenchmarkGet_EmptyCache_WriteFlavor-8                                           18083570                66.3 ns/op
BenchmarkGet_NonExistingKey_ReadFlavor-8                                        27492775                45.0 ns/op
BenchmarkGet_NonExistingKey_WriteFlavor-8                                       26621788                45.1 ns/op
BenchmarkGet_ExistingKey_ReadFlavor-8                                            4380112               235 ns/op
BenchmarkGet_ExistingKey_WriteFlavor-8                                           6341376               188 ns/op
BenchmarkSet_Parallel_ReadFlavor-8                                               1683429               678 ns/op
BenchmarkSet_Parallel_WriteFlavor-8                                              1697404               661 ns/op
BenchmarkGet_FullCache_Parallel_ReadFlavor-8                                     3324580               324 ns/op
BenchmarkGet_FullCache_Parallel_WriteFlavor-8                                    4217967               273 ns/op
BenchmarkDelete_FullCache_Parallel_ReadFlavor-8                                  4306207               253 ns/op
BenchmarkDelete_FullCache_Parallel_WriteFlavor-8                                 4283239               258 ns/op
BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_ReadFlavor-8          3930772               289 ns/op
BenchmarkDelete_FullCache_Parallel_EvictionChannelAttached_WriteFlavor-8         3913034               289 ns/op
BenchmarkKeys_EmptyCache_ReadFlavor-8                                           26115826                44.1 ns/op
BenchmarkKeys_EmptyCache_WriteFlavor-8                                          27526776                44.1 ns/op
BenchmarkKeys_FullCache_ReadFlavor-8                                                  75          19673257 ns/op
BenchmarkKeys_FullCache_WriteFlavor-8                                                 75          19900484 ns/op
BenchmarkEntries_EmptyCache_ReadFlavor-8                                        26489229                44.4 ns/op
BenchmarkEntries_EmptyCache_WriteFlavor-8                                       27718692                45.4 ns/op
BenchmarkEntries_FullCache_ReadFlavor-8                                               15          87283280 ns/op
BenchmarkEntries_FullCache_WriteFlavor-8                                              14          83859209 ns/op
```

## License

Copyright (c) 2020 Ioannis Tzanellis<br>
[Released under the MIT license](https://github.com/jahnestacado/go-tlru/blob/master/LICENSE)
