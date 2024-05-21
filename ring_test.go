package main

import (
	"fmt"
	"runtime"
	"testing"
)

const BufferSize = 2 * 1024 * 1024

func TestRingBuffer(t *testing.T) {
	tests := []struct {
		name string
		ring RingBuffer
	}{
		{"modulo", NewRingBuffer0(2)},
		{"bitmask", NewRingBuffer1(2)},
		{"mpmc with lock", NewRingBuffer2(2)},
		{"spsc with atomic", NewRingBuffer3(2)},
		{"spsc with atomic + pad", NewRingBuffer4(2)},
		{"spsc with index cache", NewRingBuffer5(2)},
		{"mpmc with atomic", NewRingBuffer7(2)},
	}

	assertNil := func(t *testing.T, err error) {
		t.Helper()
		if err != nil {
			t.Errorf("error = %v", err)
		}
	}
	assertEqual := func(t *testing.T, got, want interface{}) {
		t.Helper()
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := tt.ring.Dequeue()
			assertEqual(t, err, ErrBufferEmpty)

			err = tt.ring.Enqueue(1)
			assertNil(t, err)

			err = tt.ring.Enqueue(2)
			assertNil(t, err)

			err = tt.ring.Enqueue(3)
			assertEqual(t, err, ErrBufferFull)

			item, err = tt.ring.Dequeue()
			assertNil(t, err)
			assertEqual(t, item, 1)

			err = tt.ring.Enqueue(4)
			assertNil(t, err)

			item, err = tt.ring.Dequeue()
			assertNil(t, err)
			assertEqual(t, item, 2)

			item, err = tt.ring.Dequeue()
			assertNil(t, err)
			assertEqual(t, item, 4)

			item, err = tt.ring.Dequeue()
			assertEqual(t, err, ErrBufferEmpty)
		})
	}
}

func BenchmarkRingBufferSequential(b *testing.B) {
	runtime.GOMAXPROCS(1)
	tests := []struct {
		name string
		ring RingBuffer
	}{
		{"modulo", NewRingBuffer0(BufferSize)},
		{"bitmask", NewRingBuffer1(BufferSize)},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkRingBufferSequential(b, tt.ring)
		})
	}
}

func BenchmarkRingBuffer1P1C(b *testing.B) {
	runtime.GOMAXPROCS(2)
	tests := []struct {
		name string
		ring RingBuffer
	}{
		{"mpmc with lock", NewRingBuffer2(BufferSize)},
		{"spsc with atomic", NewRingBuffer3(BufferSize)},
		{"spsc with atomic + pad", NewRingBuffer4(BufferSize)},
		{"spsc with index cache", NewRingBuffer5(BufferSize)},
		{"mpmc with atomic (w/o pad)", NewRingBuffer7_wo_pad(BufferSize)},
		{"mpmc with atomic", NewRingBuffer7(BufferSize)},
		{"mpmc with atomic + cas spin", NewRingBuffer8(BufferSize)},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkRingBuffer(b, tt.ring, 1)
		})
	}
}
func BenchmarkRingBuffer2P1C(b *testing.B) {
	runtime.GOMAXPROCS(3)
	tests := []struct {
		name string
		ring RingBuffer
	}{
		{"mpmc with lock", NewRingBuffer2(BufferSize)},
		{"mpmc with atomic (w/o pad)", NewRingBuffer7_wo_pad(BufferSize)},
		{"mpmc with atomic", NewRingBuffer7(BufferSize)},
		{"mpmc with atomic + cas spin", NewRingBuffer8(BufferSize)},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkRingBuffer(b, tt.ring, 2)
		})
	}
}
func BenchmarkRingBuffer3P1C(b *testing.B) {
	runtime.GOMAXPROCS(4)
	tests := []struct {
		name string
		ring RingBuffer
	}{
		{"mpmc with lock", NewRingBuffer2(BufferSize)},
		{"mpmc with atomic (w/o pad)", NewRingBuffer7_wo_pad(BufferSize)},
		{"mpmc with atomic", NewRingBuffer7(BufferSize)},
		{"mpmc with atomic + cas spin", NewRingBuffer8(BufferSize)},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkRingBuffer(b, tt.ring, 3)
		})
	}
}

func benchmarkRingBufferSequential(b *testing.B, rb RingBuffer) {
	iterations := uint64(b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := uint64(0); i < iterations/1000; i++ {
		for j := 0; j < 1000; j++ {
			rb.Enqueue(i)
		}
		for j := 0; j < 1000; j++ {
			rb.Dequeue()
		}
	}
}

func benchmarkRingBuffer(b *testing.B, rb RingBuffer, writers int64) {
	iterations := int64(b.N)
	maxReads := iterations * writers

	b.ReportAllocs()
	b.ResetTimer()

	for x := int64(0); x < writers; x++ {
		go func() {
			for i := int64(0); i < iterations; {
				if rb.Enqueue(i) == nil {
					i++
				}
			}
		}()
	}

	for i := int64(0); i < maxReads; {
		msg, err := rb.Dequeue()
		if err == nil {
			if writers == 1 && msg != i {
				panic(fmt.Sprintf("Out of sequence %d %d", msg, i))
			}
			i++
		}
	}
}

func BenchmarkMPMCRingBufferCASSpin1P1C(b *testing.B) {
	benchmarkMPMCRingBufferCASSpin(b, 1)
}
func BenchmarkMPMCRingBufferCASSpin2P1C(b *testing.B) {
	benchmarkMPMCRingBufferCASSpin(b, 2)
}
func BenchmarkMPMCRingBufferCASSpin3P1C(b *testing.B) {
	benchmarkMPMCRingBufferCASSpin(b, 3)
}
func benchmarkMPMCRingBufferCASSpin(b *testing.B, writers int64) {
	iterations := int64(b.N)
	maxReads := iterations * writers
	rb := NewRingBuffer8(1024 * 16)

	b.ReportAllocs()
	b.ResetTimer()

	for x := int64(0); x < writers; x++ {
		go func() {
			for i := int64(0); i < iterations; {
				if rb.Len() == rb.Cap() {
					continue
				} else {
					rb.Enqueue(i)
					i++
				}
			}
		}()
	}

	for i := int64(0); i < maxReads; i++ {
		msg, _ := rb.Dequeue()
		if writers == 1 && msg != i {
			panic(fmt.Sprintf("Out of sequence %d %d", msg, i))
		}
	}
}
