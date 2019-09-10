package lb

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"io"
	"math"
	"math/rand"
	"testing"
)

func TestPriorityWeightedDistribution(t *testing.T) {
	resources := []Resource{
		Resource{20, 0, "d"},
		Resource{20, 0, "e"},
		Resource{10, 60, "a"},
		Resource{10, 20, "b"},
		Resource{10, 20, "c"},
	}

	lb, _ := NewLoadBalancer(resources)

	const (
		n     = 1.0e7
		delta = 1.0e-3
	)

	// Distribution of 2 targets with equal weights.
	dist := make([]float64, 2)
	expected := []float64{0.5, 0.5}
	for i := 0; i < int(n); i++ {
		target, _, _ := lb.PriorityWeighted(11)

		// d-e mapped to 0-1
		dist[target.(string)[0]-'d'] += 1.0
	}

	for i := range dist {
		if diff := math.Abs(dist[i]/n - expected[i]); diff > delta {
			t.Fatalf("dist[%d] of 2, delta is %f", i, diff)
		}
	}

	// Distribution of 3 targets with differing weights.
	dist = make([]float64, 3)
	expected = []float64{0.6, 0.2, 0.2}
	for i := 0; i < int(n); i++ {
		target, _, _ := lb.PriorityWeighted(0)

		// a-c mapped to 0-2
		dist[target.(string)[0]-'a'] += 1.0
	}

	for i := range dist {
		if diff := math.Abs(dist[i]/n - expected[i]); diff > delta {
			t.Fatalf("dist[%d] of 3, delta is %f", i, diff)
		}
	}
}

func TestIgnoreZeroWeight(t *testing.T) {
	resources := []Resource{
		Resource{0, 0, "a"},
		Resource{0, 1, "b"},
		Resource{0, 0, "c"},
		Resource{0, 1, "d"},
	}

	lb, _ := NewLoadBalancer(resources)

	for i := 0; i < 100; i++ {
		target, _, err := lb.PriorityWeighted(0)

		if err != nil {
			t.Fatal(err)
		} else if got := target.(string); got != "b" && got != "d" {
			t.Fatalf("got target '%s' instead of 'b'", got)
		}
	}
}

func BenchmarkPriorityWeightedDefaultRand(b *testing.B) {
	resources := []Resource{
		Resource{20, 0, "d"},
		Resource{20, 0, "e"},
		Resource{10, 60, "a"},
		Resource{10, 20, "b"},
		Resource{10, 20, "c"},
	}

	lb, _ := NewLoadBalancer(resources)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = lb.PriorityWeighted(0)
	}
}

type source struct {
	r io.Reader
}

func (s source) Int63() int64 { return int64(s.UInt64() & (1<<63 - 1)) }
func (s source) Seed(int64)   {}
func (s source) UInt64() uint64 {
	var buf [8]byte
	_, _ = io.ReadFull(s.r, buf[:])
	return binary.LittleEndian.Uint64(buf[:])
}

// Using full cryptographic randomness is expected to introduce a ~25% slowdown.
func BenchmarkPriorityWeightedCryptoRand(b *testing.B) {
	resources := []Resource{
		Resource{20, 0, "d"},
		Resource{20, 0, "e"},
		Resource{10, 60, "a"},
		Resource{10, 20, "b"},
		Resource{10, 20, "c"},
	}

	lb, _ := NewLoadBalancer(resources)
	lb.Rand = rand.New(source{cryptorand.Reader})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = lb.PriorityWeighted(0)
	}
}
