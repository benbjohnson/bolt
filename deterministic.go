package bolt

import (
	"math/rand"
	"sort"
)

// DETERMINISTIC:
func randomizeBucketMapKeys(m map[string]*Bucket) []string {
	a := make([]string, 0, len(m))
	for name := range m {
		a = append(a, name)
	}
	sort.Strings(a)
	randomizeStrings(a)
	return a
}

func randomizeNodeMapKeys(m map[pgid]*node) []pgid {
	a := make([]pgid, 0, len(m))
	for id := range m {
		a = append(a, id)
	}
	sort.Sort(pgids(a))
	randomizePgids(a)
	return a
}

func randomizePageMapKeys(m map[pgid]*page) []pgid {
	a := make([]pgid, 0, len(m))
	for id := range m {
		a = append(a, id)
	}
	sort.Sort(pgids(a))
	randomizePgids(a)
	return a
}

func randomizePendingMapKeys(m map[txid][]pgid) []txid {
	a := make([]txid, 0, len(m))
	for id := range m {
		a = append(a, id)
	}
	sort.Sort(txids(a))
	randomizeTxids(a)
	return a
}

func randomizeStrings(a []string) []string {
	other := make([]string, len(a))
	for i, j := range newRand().Perm(len(a)) {
		other[j] = a[i]
	}
	return other
}

func randomizePgids(a []pgid) []pgid {
	other := make([]pgid, len(a))
	for i, j := range newRand().Perm(len(a)) {
		other[j] = a[i]
	}
	return other
}

func randomizeTxids(a []txid) []txid {
	other := make([]txid, len(a))
	for i, j := range newRand().Perm(len(a)) {
		other[j] = a[i]
	}
	return other
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(0))
}
