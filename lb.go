// Package lb provides a simple and efficient load balancer.
//
// There are four different ways to select a resource depending on whether you
// wish to use the resource priority or weight.
//
//	┌───────────────────────┬─────────────────────┬───────────────────┐
//	│                       │ Weighted resources  │    Any weight     │
//	├───────────────────────┼─────────────────────┼───────────────────┤
//	│ Prioritized resources │ PriorityWeighted(n) │ PriorityRandom(n) │
//	├───────────────────────┼─────────────────────┼───────────────────┤
//	│ Any priority          │ RandomWeighted()    │ Random()          │
//	└───────────────────────┴─────────────────────┴───────────────────┘
package lb

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
)

// Resource represents a single balance-able resource.
type Resource struct {
	Priority uint64
	Weight   uint64
	Target   interface{}
}

type group struct {
	items []Resource
	sums  []uint64
	max   int64
}

// LoadBalancer represents a series of resources.
type LoadBalancer struct {
	// Rand is used when selecting random resources. By default Rand is
	// initialized by NewLoadBalancer to use "math/rand" with a
	// cryptographically random seed.
	Rand *rand.Rand

	groups []group
}

type byPriority []Resource

func (p byPriority) Len() int           { return len(p) }
func (p byPriority) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p byPriority) Less(i, j int) bool { return p[i].Priority < p[j].Priority }

// NewLoadBalancer constructs a load balancer from the provided resources. The
// resources are grouped by priority.
//
// If all resources within a priority group have a weight of zero they will have
// uniform weight; if a resource group has some zero and nonzero resources the
// zero-weight resources will be ignored when weight is used.
func NewLoadBalancer(resources []Resource) (*LoadBalancer, error) {
	if len(resources) == 0 {
		return nil, errors.New("balance: empty resources")
	}

	sort.Sort(byPriority(resources))

	lb := LoadBalancer{
		groups: make([]group, 1),
	}

	last := resources[0].Priority
	i := 0
	for _, resource := range resources {
		if resource.Priority != last {
			i++
			last = resource.Priority
			lb.groups = append(lb.groups, group{})
		}

		lb.groups[i].items = append(lb.groups[i].items, resource)
	}

	for i := range lb.groups {
		lb.groups[i].sums = make([]uint64, len(lb.groups[i].items))

		var total uint64

		for j, item := range lb.groups[i].items {
			total += item.Weight
			lb.groups[i].sums[j] = total
		}

		// If all weights are zero, set all weights to one to preserve
		// random distribution.
		if total == 0 {
			for j := range lb.groups[i].items {
				lb.groups[i].items[j].Weight = 1
				lb.groups[i].sums[j] = uint64(j) + 1
			}

			total = uint64(len(lb.groups[i].items))
		}

		lb.groups[i].max = int64(total)
	}

	// Seed b.Rand nondeterministically.
	seed, err := cryptorand.Int(cryptorand.Reader,
		big.NewInt(int64(^uint64(0)>>1)))

	if err != nil {
		return nil, err
	}

	lb.Rand = rand.New(rand.NewSource(seed.Int64()))

	return &lb, nil
}

// PriorityRandom finds the first resource group with priority >= n, and selects
// a random resource within that group using a uniform distribution.
//
// Returns the resource's target and priority.
func (lb *LoadBalancer) PriorityRandom(n uint64) (interface{}, uint64, error) {
	g, err := lb.fromPriority(n)

	if err != nil {
		return nil, 0, err
	}

	r := g.items[lb.Rand.Int63n(int64(len(g.items)))]

	return r.Target, r.Priority, nil
}

// PriorityWeighted finds the first resource group with priority >= n, and uses
// the resource group's weight distribution to select a resource.
//
// Returns the resource's target and priority.
func (lb *LoadBalancer) PriorityWeighted(n uint64) (interface{}, uint64, error) {
	g, err := lb.fromPriority(n)

	if err != nil {
		return nil, 0, err
	}

	r, err := lb.fromWeight(g)

	if err != nil {
		return nil, 0, err
	}

	return r.Target, r.Priority, nil
}

// Random selects a random resource using a uniform distribution, without
// respect for priority nor weight.
//
// Returns the resource's target.
func (lb *LoadBalancer) Random() (interface{}, error) {
	g := &lb.groups[lb.Rand.Int63n(int64(len(lb.groups)))]
	return g.items[lb.Rand.Int63n(int64(len(g.items)))].Target, nil
}

// RandomWeighted finds a random resource group and uses the resource group's
// weight distribution to select a resource.
//
// Returns the resource's target.
func (lb *LoadBalancer) RandomWeighted() (interface{}, error) {
	g := &lb.groups[lb.Rand.Int63n(int64(len(lb.groups)))]
	return lb.fromWeight(g)
}

func (lb *LoadBalancer) fromPriority(n uint64) (*group, error) {
	for i := range lb.groups {
		if lb.groups[i].items[0].Priority >= n {
			return &lb.groups[i], nil
		}
	}

	return nil, fmt.Errorf("balance: no resources with priority >= %d", n)
}

func (lb *LoadBalancer) fromWeight(g *group) (*Resource, error) {
	if len(g.items) == 1 {
		return &g.items[0], nil
	}

	n := uint64(lb.Rand.Int63n(g.max))

	for i, weight := range g.sums {
		if n < weight && (i == 0 || n >= g.sums[i-1]) {
			return &g.items[i], nil
		}
	}

	return nil, fmt.Errorf("balance: unable to find resource")
}
