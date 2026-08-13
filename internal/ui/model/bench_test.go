package model

import (
	"testing"

	"lazypost/internal/collection"
	"lazypost/internal/session"
)

// BenchmarkNewModel measures model construction on the real sample
// collection — load + New, the non-I/O startup path (rendering and
// terminal I/O are excluded by design).
func BenchmarkNewModel(b *testing.B) {
	entries, err := collection.Load("../../../sample-collections")
	if err != nil {
		b.Fatal(err)
	}
	envs, names, err := collection.LoadEnvironments("../../../sample-collections")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New("../../../sample-collections", entries, envs, names, session.State{})
	}
}
