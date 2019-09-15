// Package rr provides round-robins.
package rr

import "sync"

// RoundRobin represents a cyclic buffer of items.
type RoundRobin interface {
	// Next retrieves the next item.
	Next() interface{}

	// Skip skips n items.
	Skip(n int)
}

type defaultRR struct {
	items List
	index int
}

// List represents anything which can be indexed and has a determinable length.
// Most useful as a wrapper around statically-typed arrays, but can be used with
// any kind of list data type.
type List interface {
	// At retrieves the element at index n.
	At(n int) interface{}

	// Len returns the length of the list.
	Len() int
}

// NewRoundRobin creates a new round-robin buffer.
//
// If you need to call it concurrently, use NewLockedRoundRobin instead.
func NewRoundRobin(items List) RoundRobin {
	return RoundRobin(&defaultRR{items: items})
}

func (r *defaultRR) Next() interface{} {
	if r.items.Len() == 0 {
		return nil
	}

	if r.index >= r.items.Len() {
		r.index = 0
	}

	r.index++
	return r.items.At(r.index - 1)
}

func (r *defaultRR) Skip(n int) {
	if l := r.items.Len(); l != 0 {
		r.index += n % l
	}
}

type lockedRR struct {
	base RoundRobin
	mu   sync.Mutex
}

// NewLockedRoundRobin creates a synchronized round-robin buffer using mutexes.
func NewLockedRoundRobin(items List) RoundRobin {
	return Locked(NewRoundRobin(items))
}

// Locked wraps r with mutexes. Panics if r has already been locked.
func Locked(r RoundRobin) RoundRobin {
	if _, ok := r.(*lockedRR); ok {
		panic("rr: wrapped an already-synchronized round-robin")
	}

	return RoundRobin(&lockedRR{base: r})
}

func (r *lockedRR) Next() interface{} {
	r.mu.Lock()
	ret := r.base.Next()
	r.mu.Unlock()
	return ret
}

func (r *lockedRR) Skip(n int) {
	r.mu.Lock()
	r.base.Skip(n)
	r.mu.Unlock()
}
