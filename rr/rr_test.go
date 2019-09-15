package rr

import (
	"sync"
	"testing"
)

type intList []int

func (i intList) At(n int) interface{} { return i[n] }
func (i intList) Len() int             { return len(i) }

func TestPlain(t *testing.T) {
	cases := []struct {
		times int
		skip  int
		have  []int
		want  []int
	}{
		{1, 0, nil, nil},
		{1, 1, nil, nil},
		{1, 0, []int{}, nil},
		{1, 1, []int{}, nil},
		{10, 0, []int{0, 1, 2}, []int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0}},
	}

	for i, c := range cases {
		r := NewRoundRobin(intList(c.have))
		r.Skip(c.skip)

		for j := range c.want {
			if have := r.Next().(int); have != c.want[j] {
				t.Fatalf("case %d: want %d have %d at index %d",
					i, c.want[j], have, j)
			}
		}
	}
}

// Test with -race
func TestLocked(t *testing.T) {
	items := []int{0, 1, 2, 3, 4}
	const count = 1000

	r := NewLockedRoundRobin(intList(items))

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(r RoundRobin) {
			r.Next()
			wg.Done()
		}(r)
	}

	wg.Wait()
}

func BenchmarkPlain(b *testing.B) {
	r := NewRoundRobin(intList([]int{0, 1, 2, 3, 4, 5}))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Next()
	}
}

// Since the List is an array, having a huge list should perform no different
// than a small list.
func BenchmarkPlainHuge(b *testing.B) {
	list := make([]int, 1e4)

	for i := range list {
		list[i] = i
	}

	r := NewRoundRobin(intList(list))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Next()
	}
}

func BenchmarkLocked(b *testing.B) {
	r := NewLockedRoundRobin(intList([]int{0, 1, 2, 3, 4, 5}))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Next()
	}
}
