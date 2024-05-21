package main

import (
	"fmt"
	"testing"
)

func BenchmarkChannelBlocking1P1C(b *testing.B) {
	benchmarkBlocking(b, 1)
}
func BenchmarkChannelBlocking2P1C(b *testing.B) {
	benchmarkBlocking(b, 2)
}
func BenchmarkChannelBlocking3P1C(b *testing.B) {
	benchmarkBlocking(b, 3)
}
func benchmarkBlocking(b *testing.B, writers int64) {
	channel := make(chan int64, 1024*16)
	iterations := int64(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for x := int64(0); x < writers; x++ {
		go func() {
			for i := int64(0); i < iterations; i++ {
				channel <- i
			}
		}()
	}

	for i := int64(0); i < iterations*writers; i++ {
		msg := <-channel
		if writers == 1 && msg != i {
			panic(fmt.Sprintf("Out of sequence %d %d", msg, i))
		}
	}
}

func BenchmarkChannelNonBlocking1P1C(b *testing.B) {
	benchmarkNonBlocking(b, 1)
}
func BenchmarkChannelNonBlocking2P1C(b *testing.B) {
	benchmarkNonBlocking(b, 2)
}
func BenchmarkChannelNonBlocking3P1C(b *testing.B) {
	benchmarkNonBlocking(b, 3)
}
func benchmarkNonBlocking(b *testing.B, writers int64) {
	iterations := int64(b.N)
	maxReads := iterations * writers
	channel := make(chan int64, 1024*16)

	b.ReportAllocs()
	b.ResetTimer()

	for x := int64(0); x < writers; x++ {
		go func() {
			for i := int64(0); i < iterations; {
				select {
				case channel <- i:
					i++
				default:
					continue
				}
			}
		}()
	}

	for i := int64(0); i < maxReads; {
		select {
		case msg := <-channel:
			if writers == 1 && msg != i {
				panic(fmt.Sprintf("Out of sequence %d %d", msg, i))
			}
			i++
		default:
			continue
		}
	}
}
