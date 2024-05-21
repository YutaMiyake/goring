package main

import (
	"errors"
	"golang.org/x/sys/cpu"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	ErrBufferFull  = errors.New("buffer full")
	ErrBufferEmpty = errors.New("buffer empty")
)

// CacheLinePadSize in your system
const CacheLinePadSize = unsafe.Sizeof(cpu.CacheLinePad{})

type RingBuffer interface {
	Enqueue(item any) error
	Dequeue() (any, error)
}

// baseline (single-thread)

type RingBuffer0 struct {
	writeIdx uint64
	readIdx  uint64
	size     uint64
	buffers  []any
}

func NewRingBuffer0(size uint64) *RingBuffer0 {
	rb := &RingBuffer0{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer0) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer0) Enqueue(item any) error {
	if rb.writeIdx-rb.readIdx == rb.size {
		return ErrBufferFull
	}
	rb.buffers[rb.writeIdx%rb.size] = item
	rb.writeIdx++
	return nil
}

func (rb *RingBuffer0) Dequeue() (any, error) {
	if rb.writeIdx == rb.readIdx {
		return nil, ErrBufferEmpty
	}
	item := rb.buffers[rb.readIdx%rb.size]
	rb.readIdx++
	return item, nil
}

// bitmask

type RingBuffer1 struct {
	writeIdx uint64
	readIdx  uint64
	size     uint64
	mask     uint64
	buffers  []any
}

func NewRingBuffer1(size uint64) *RingBuffer1 {
	rb := &RingBuffer1{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer1) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
	rb.mask = size - 1
}

func (rb *RingBuffer1) Enqueue(item any) error {
	if rb.writeIdx-rb.readIdx == rb.size {
		return ErrBufferFull
	}
	rb.buffers[rb.writeIdx&rb.mask] = item
	rb.writeIdx++
	return nil
}

func (rb *RingBuffer1) Dequeue() (any, error) {
	if rb.writeIdx == rb.readIdx {
		return nil, ErrBufferEmpty
	}
	item := rb.buffers[rb.readIdx&rb.mask]
	rb.readIdx++
	return item, nil
}

// mpmc, mutex

type RingBuffer2 struct {
	writeIdx uint64
	readIdx  uint64
	size     uint64
	buffers  []any
	mu       sync.Mutex
}

func NewRingBuffer2(size uint64) *RingBuffer2 {
	rb := &RingBuffer2{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer2) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer2) Enqueue(item any) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.writeIdx-rb.readIdx == rb.size {
		return ErrBufferFull
	}
	rb.buffers[rb.writeIdx&(rb.size-1)] = item
	rb.writeIdx++
	return nil
}

func (rb *RingBuffer2) Dequeue() (any, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	if rb.writeIdx == rb.readIdx {
		return nil, ErrBufferEmpty
	}
	item := rb.buffers[rb.readIdx&(rb.size-1)]
	rb.readIdx++
	return item, nil
}

// spsc, atomic

type RingBuffer3 struct {
	writeIdx uint64
	readIdx  uint64
	size     uint64
	buffers  []any
}

func NewRingBuffer3(size uint64) *RingBuffer3 {
	rb := &RingBuffer3{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer3) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer3) Enqueue(item any) error {
	read := atomic.LoadUint64(&rb.readIdx)
	write := rb.writeIdx
	if write == read+rb.size {
		runtime.Gosched()
		return ErrBufferFull
	}
	rb.buffers[write&(rb.size-1)] = item
	atomic.StoreUint64(&rb.writeIdx, write+1)
	return nil
}

func (rb *RingBuffer3) Dequeue() (any, error) {
	write := atomic.LoadUint64(&rb.writeIdx)
	read := rb.readIdx
	if write == read {
		runtime.Gosched()
		return nil, ErrBufferEmpty
	}
	item := rb.buffers[read&(rb.size-1)]
	atomic.StoreUint64(&rb.readIdx, read+1)
	return item, nil
}

// padding

type RingBuffer4 struct {
	writeIdx uint64
	_        [CacheLinePadSize]byte
	readIdx  uint64
	_        [CacheLinePadSize]byte
	size     uint64
	buffers  []any
}

func NewRingBuffer4(size uint64) *RingBuffer4 {
	rb := &RingBuffer4{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer4) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer4) Enqueue(item any) error {
	read := atomic.LoadUint64(&rb.readIdx)
	write := rb.writeIdx
	if write == read+rb.size {
		runtime.Gosched()
		return ErrBufferFull
	}
	rb.buffers[write&(rb.size-1)] = item
	atomic.StoreUint64(&rb.writeIdx, write+1)
	return nil
}

func (rb *RingBuffer4) Dequeue() (any, error) {
	write := atomic.LoadUint64(&rb.writeIdx)
	read := rb.readIdx
	if write == read {
		runtime.Gosched()
		return nil, ErrBufferEmpty
	}
	item := rb.buffers[read&(rb.size-1)]
	atomic.StoreUint64(&rb.readIdx, read+1)
	return item, nil
}

// index cache
// based on https://kumagi.hatenablog.com/entry/ring-buffer

type RingBuffer5 struct {
	writeIdx   uint64
	_          [CacheLinePadSize]byte
	readIdx    uint64
	_          [CacheLinePadSize]byte
	writeCache uint64
	_          [CacheLinePadSize]byte
	readCache  uint64
	_          [CacheLinePadSize]byte
	size       uint64
	_          [CacheLinePadSize]byte
	buffers    []any
}

func NewRingBuffer5(size uint64) *RingBuffer5 {
	rb := &RingBuffer5{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer5) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer5) Enqueue(item any) error {
	write := rb.writeIdx
	if write == rb.readCache+rb.size {
		rb.readCache = atomic.LoadUint64(&rb.readIdx)
		if write == rb.readCache+rb.size {
			runtime.Gosched()
			return ErrBufferFull
		}
	}
	rb.buffers[write&(rb.size-1)] = item
	atomic.StoreUint64(&rb.writeIdx, write+1)
	return nil
}

func (rb *RingBuffer5) Dequeue() (any, error) {
	read := rb.readIdx
	if rb.writeCache == read {
		rb.writeCache = atomic.LoadUint64(&rb.writeIdx)
		if rb.writeCache == read {
			runtime.Gosched()
			return nil, ErrBufferEmpty
		}
	}
	item := rb.buffers[read&(rb.size-1)]
	atomic.StoreUint64(&rb.readIdx, read+1)
	return item, nil
}

// index cache with spin (block on full/empty)
// based on https://kumagi.hatenablog.com/entry/ring-buffer

type RingBuffer6 struct {
	writeIdx   uint64
	_          [CacheLinePadSize]byte
	readIdx    uint64
	_          [CacheLinePadSize]byte
	writeCache uint64
	_          [CacheLinePadSize]byte
	readCache  uint64
	_          [CacheLinePadSize]byte
	size       uint64
	_          [CacheLinePadSize]byte
	buffers    []any
}

func NewRingBuffer6(size uint64) *RingBuffer6 {
	rb := &RingBuffer6{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer6) init(size uint64) {
	rb.buffers = make([]any, size)
	rb.size = size
}

func (rb *RingBuffer6) Enqueue(item any) error {
	read := rb.readCache
	write := rb.writeIdx
	for write == read+rb.size {
		rb.readCache = atomic.LoadUint64(&rb.readIdx)
		read = rb.readCache
		runtime.Gosched()
	}
	rb.buffers[write&(rb.size-1)] = item
	atomic.StoreUint64(&rb.writeIdx, write+1)
	return nil
}

func (rb *RingBuffer6) Dequeue() (any, error) {
	write := rb.writeCache
	read := rb.readIdx
	for write == read {
		rb.writeCache = atomic.LoadUint64(&rb.writeIdx)
		write = rb.writeCache
		runtime.Gosched()
	}
	item := rb.buffers[read&(rb.size-1)]
	atomic.StoreUint64(&rb.readIdx, read+1)
	return item, nil
}

// mpmc, atomic
// based on https://github.com/Workiva/go-datastructures/blob/v1.1.4/queue/ring.go

type node_wo_pad struct {
	position uint64
	data     any
}

type nodes_wo_pad []node

// without pad

type RingBuffer7_wo_pad struct {
	writeIdx uint64
	readIdx  uint64
	mask     uint64
	nodes    nodes_wo_pad
}

func NewRingBuffer7_wo_pad(size uint64) *RingBuffer7_wo_pad {
	rb := &RingBuffer7_wo_pad{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer7_wo_pad) init(size uint64) {
	rb.nodes = make(nodes_wo_pad, size)
	for i := uint64(0); i < size; i++ {
		rb.nodes[i] = node{position: i}
	}
	rb.mask = size - 1
}

func (rb *RingBuffer7_wo_pad) Enqueue(item interface{}) error {
	write := atomic.LoadUint64(&rb.writeIdx)
	n := &rb.nodes[write&rb.mask]
	seq := atomic.LoadUint64(&n.position)
	if seq != write {
		return ErrBufferFull
	}
	if !atomic.CompareAndSwapUint64(&rb.writeIdx, write, write+1) {
		runtime.Gosched()
		return ErrBufferFull
	}

	n.data = item
	atomic.StoreUint64(&n.position, write+1)
	return nil
}

func (rb *RingBuffer7_wo_pad) Dequeue() (interface{}, error) {
	read := atomic.LoadUint64(&rb.readIdx)
	n := &rb.nodes[read&rb.mask]
	seq := atomic.LoadUint64(&n.position)
	if seq != read+1 {
		return nil, ErrBufferEmpty
	}
	if !atomic.CompareAndSwapUint64(&rb.readIdx, read, read+1) {
		runtime.Gosched()
		return nil, ErrBufferEmpty
	}

	data := n.data
	n.data = nil
	atomic.StoreUint64(&n.position, read+rb.mask+1)
	return data, nil
}

type node struct {
	position uint64
	data     any
	_        [CacheLinePadSize]byte
}

type nodes []node

type RingBuffer7 struct {
	_        [CacheLinePadSize]byte
	writeIdx uint64
	_        [CacheLinePadSize]byte
	readIdx  uint64
	_        [CacheLinePadSize]byte
	mask     uint64
	_        [CacheLinePadSize]byte
	nodes    nodes
}

func NewRingBuffer7(size uint64) *RingBuffer7 {
	rb := &RingBuffer7{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer7) init(size uint64) {
	rb.nodes = make(nodes, size)
	for i := uint64(0); i < size; i++ {
		rb.nodes[i] = node{position: i}
	}
	rb.mask = size - 1
}

func (rb *RingBuffer7) Enqueue(item interface{}) error {
	write := atomic.LoadUint64(&rb.writeIdx)
	n := &rb.nodes[write&rb.mask] //
	seq := atomic.LoadUint64(&n.position)
	if seq != write {
		return ErrBufferFull
	}
	if !atomic.CompareAndSwapUint64(&rb.writeIdx, write, write+1) {
		runtime.Gosched()
		return ErrBufferFull
	}

	n.data = item
	atomic.StoreUint64(&n.position, write+1)
	return nil
}

func (rb *RingBuffer7) Dequeue() (interface{}, error) {
	read := atomic.LoadUint64(&rb.readIdx)
	n := &rb.nodes[read&rb.mask]
	seq := atomic.LoadUint64(&n.position)
	if seq != read+1 {
		return nil, ErrBufferEmpty
	}
	if !atomic.CompareAndSwapUint64(&rb.readIdx, read, read+1) {
		runtime.Gosched()
		return nil, ErrBufferEmpty
	}

	data := n.data
	n.data = nil
	atomic.StoreUint64(&n.position, read+rb.mask+1)
	return data, nil
}

// mpmc, atomic, CAS spin (block on full/empty)
// based on https://github.com/Workiva/go-datastructures/blob/v1.1.4/queue/ring.go

type RingBuffer8 struct {
	_        [CacheLinePadSize]byte
	writeIdx uint64
	_        [CacheLinePadSize]byte
	readIdx  uint64
	_        [CacheLinePadSize]byte
	mask     uint64
	_        [CacheLinePadSize]byte
	nodes    nodes
}

func NewRingBuffer8(size uint64) *RingBuffer8 {
	rb := &RingBuffer8{}
	rb.init(size)
	return rb
}

func (rb *RingBuffer8) init(size uint64) {
	rb.nodes = make(nodes, size)
	for i := uint64(0); i < size; i++ {
		rb.nodes[i] = node{position: i}
	}
	rb.mask = size - 1
}

func (rb *RingBuffer8) Enqueue(item interface{}) error {
	var n *node
	write := atomic.LoadUint64(&rb.writeIdx)
L:
	for {
		n = &rb.nodes[write&rb.mask]
		seq := atomic.LoadUint64(&n.position)
		switch dif := seq - write; {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&rb.writeIdx, write, write+1) {
				break L
			}
		default:
			write = atomic.LoadUint64(&rb.writeIdx)
		}

		runtime.Gosched()
	}

	n.data = item
	atomic.StoreUint64(&n.position, write+1)
	return nil
}

func (rb *RingBuffer8) Dequeue() (interface{}, error) {
	var n *node
	read := atomic.LoadUint64(&rb.readIdx)
L:
	for {
		n = &rb.nodes[read&rb.mask]
		seq := atomic.LoadUint64(&n.position)
		switch dif := seq - (read + 1); {
		case dif == 0:
			if atomic.CompareAndSwapUint64(&rb.readIdx, read, read+1) {
				break L
			}
		default:
			read = atomic.LoadUint64(&rb.readIdx)
		}
		runtime.Gosched()
	}
	data := n.data
	n.data = nil
	atomic.StoreUint64(&n.position, read+rb.mask+1)
	return data, nil
}

func (rb *RingBuffer8) Cap() uint64 {
	return uint64(len(rb.nodes))
}

func (rb *RingBuffer8) Len() uint64 {
	return atomic.LoadUint64(&rb.writeIdx) - atomic.LoadUint64(&rb.readIdx)
}
