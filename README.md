Benchmarking: RingBuffer Implementations

Environment
```
Apple M1 Pro
go version go1.21.4 darwin/arm64
```

Benchmarks
```
❯ go test -test.bench .
goos: darwin
goarch: arm64
pkg: ringbuffer
BenchmarkChannelBlocking1P1C-10                  6251745               168.6 ns/op             0 B/op          0 allocs/op
BenchmarkChannelBlocking2P1C-10                  6269666               181.8 ns/op             0 B/op          0 allocs/op
BenchmarkChannelBlocking3P1C-10                  6148294               198.8 ns/op             0 B/op          0 allocs/op
BenchmarkChannelNonBlocking1P1C-10               6500550               179.0 ns/op             0 B/op          0 allocs/op
BenchmarkChannelNonBlocking2P1C-10               5435923               248.6 ns/op             0 B/op          0 allocs/op
BenchmarkChannelNonBlocking3P1C-10               2834624               358.0 ns/op             0 B/op          0 allocs/op
BenchmarkRingBufferSequential/modulo-10         100000000               11.94 ns/op            7 B/op          0 allocs/op
BenchmarkRingBufferSequential/bitmask-10        100000000               12.03 ns/op            7 B/op          0 allocs/op
BenchmarkRingBuffer1P1C/mpmc_with_lock-10               15724068               123.1 ns/op             7 B/op          0 allocs/op
BenchmarkRingBuffer1P1C/spsc_with_atomic-10             14667102                87.55 ns/op            7 B/op          0 allocs/op
BenchmarkRingBuffer1P1C/spsc_with_atomic_+_pad-10        5960235               227.3 ns/op             7 B/op          0 allocs/op
BenchmarkRingBuffer1P1C/spsc_with_index_cache-10        99715047                14.33 ns/op            7 B/op          0 allocs/op
BenchmarkRingBuffer1P1C/mpmc_with_atomic_(w/o_pad)-10            6838928               207.2 ns/op             7 B/op        0 allocs/op
BenchmarkRingBuffer1P1C/mpmc_with_atomic-10                     24418657                88.10 ns/op            7 B/op        0 allocs/op
BenchmarkRingBuffer1P1C/mpmc_with_atomic_+_cas_spin-10          23752008                87.10 ns/op            7 B/op        0 allocs/op
BenchmarkRingBuffer2P1C/mpmc_with_lock-10                        4458638               364.4 ns/op            23 B/op        2 allocs/op
BenchmarkRingBuffer2P1C/mpmc_with_atomic_(w/o_pad)-10            2810728               436.8 ns/op            24 B/op        3 allocs/op
BenchmarkRingBuffer2P1C/mpmc_with_atomic-10                      3892406               307.6 ns/op            23 B/op        2 allocs/op
BenchmarkRingBuffer2P1C/mpmc_with_atomic_+_cas_spin-10          21175022                54.28 ns/op           15 B/op        1 allocs/op
BenchmarkRingBuffer3P1C/mpmc_with_lock-10                        2546384              1018 ns/op              51 B/op        6 allocs/op
BenchmarkRingBuffer3P1C/mpmc_with_atomic_(w/o_pad)-10            2154146               469.3 ns/op            33 B/op        4 allocs/op
BenchmarkRingBuffer3P1C/mpmc_with_atomic-10                      6168470               191.5 ns/op            29 B/op        3 allocs/op
BenchmarkRingBuffer3P1C/mpmc_with_atomic_+_cas_spin-10          12486115                95.95 ns/op           23 B/op        2 allocs/op
BenchmarkMPMCRingBufferCASSpin1P1C-10                            8488903               190.3 ns/op             7 B/op        0 allocs/op
BenchmarkMPMCRingBufferCASSpin2P1C-10                            6757401               179.4 ns/op            15 B/op        1 allocs/op
BenchmarkMPMCRingBufferCASSpin3P1C-10                            5932155               198.4 ns/op            23 B/op        2 allocs/op
PASS
ok      ringbuffer      44.219s

```
