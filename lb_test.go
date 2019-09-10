package lb

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"io"
	"math"
	"math/rand"
	"testing"
)

// TestWeightedDistribution checks that the calculation of resource weights
// preserves their intended distribution.
func TestWeightedDistribution(t *testing.T) {
	const (
		n     = 1.0e7
		delta = 1.0e-3
	)

	distParams := []distParams{
		distParams{
			resources: []Resource{
				Resource{20, 0, 0},
				Resource{20, 0, 1},
			},
			expected: []float64{0.5, 0.5},
		},
		distParams{
			resources: []Resource{
				Resource{10, 60, 0},
				Resource{10, 10, 1},
				Resource{10, 30, 2},
			},
			expected: []float64{0.6, 0.1, 0.3},
		},
		distParams{
			resources: []Resource{
				Resource{0, 0, 0},
				Resource{0, 1, 1},
				Resource{0, 0, 2},
				Resource{0, 1, 3},
			},
			expected: []float64{0.0, 0.5, 0.0, 0.5},
		},
	}

	for _, params := range distParams {
		distribution(t, n, delta, params)
	}
}

type distParams struct {
	resources []Resource
	expected  []float64
}

// Check distribution of resources according to expected averages. If expected
// average is nonzero, delta will be used to check proximity.
func distribution(t *testing.T, n float64, delta float64, p distParams) {
	lb, _ := NewLoadBalancer(p.resources)
	dist := make([]float64, len(p.expected))

	for i := 0; i < int(n); i++ {
		target, _, _ := lb.PriorityWeighted(0)
		dist[target.(int)] += 1.0
	}

	for i := range dist {
		avg := dist[i] / n

		if p.expected[i] == 0.0 && avg != 0.0 {
			t.Fatalf("dist[%d] has %f expected zero", i, avg)
		}

		if diff := math.Abs(avg - p.expected[i]); diff > delta {
			t.Fatalf("dist[%d] has delta %f near %f", i, diff,
				p.expected[i])
		}
	}
}

// Common resources used to benchmark selection functions.
var benchmarkResources = []Resource{
	Resource{1, 60, 0},
	Resource{1, 20, 0},
	Resource{1, 20, 0},
	Resource{2, 60, 0},
	Resource{2, 40, 0},
	Resource{3, 80, 0},
	Resource{3, 20, 0},
}

// Load balancer used to benchmark selection functions.
var benchmarkLb, _ = NewLoadBalancer(benchmarkResources)

// BenchmarkPriorityRandom to access the middle resource group.
func BenchmarkPriorityRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = benchmarkLb.PriorityRandom(2)
	}
}

// BenchmarkPriorityWeighted to access the middle resource group.
func BenchmarkPriorityWeighted(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = benchmarkLb.PriorityWeighted(2)
	}
}

func BenchmarkRandom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = benchmarkLb.Random()
	}
}

func BenchmarkRandomWeighted(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = benchmarkLb.RandomWeighted()
	}
}

type source struct {
	r io.Reader
}

func (s source) Int63() int64 { return int64(s.UInt64() & (1<<63 - 1)) }
func (s source) Seed(int64)   {}
func (s source) UInt64() uint64 {
	// buf will be dynamically allocated, which introduces a 25x slowdown.
	// Using a global variable with mutexes is about just as slow. Using an
	// un-mutexed global variable is only 2x slower than normal "math/rand"
	// but will have race conditions.
	var buf [8]byte
	_, _ = io.ReadFull(s.r, buf[:])
	return binary.LittleEndian.Uint64(buf[:])
}

// BenchmarkCryptoRand benchmarks PriorityWeighted using the "crypto/rand"
// source. Using full cryptographic randomness is much slower.
func BenchmarkCryptoRand(b *testing.B) {
	lb, _ := NewLoadBalancer(benchmarkResources)
	lb.Rand = rand.New(source{cryptorand.Reader})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = lb.PriorityWeighted(2)
	}
}
