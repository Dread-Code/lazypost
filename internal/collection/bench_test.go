package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// benchTree builds a temp collection of 100 requests across 10 folders
// plus one environment, mirroring a real collection layout.
func benchTree(b *testing.B) string {
	b.Helper()
	root := b.TempDir()
	for i := 0; i < 10; i++ {
		dir := filepath.Join(root, fmt.Sprintf("folder%02d", i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 10; j++ {
			body := fmt.Sprintf("name: request %d-%d\nmethod: GET\nurl: https://api.test/f%02d/r%02d\n", i, j, i, j)
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("r%02d.yaml", j)), []byte(body), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	envDir := filepath.Join(root, "environments")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "dev.yaml"), []byte("variables:\n  host: https://api.test\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	return root
}

// BenchmarkLoad measures the collection tree walk + YAML parse — the
// startup parse path (main.go).
func BenchmarkLoad(b *testing.B) {
	root := benchTree(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(root); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadEnvironments measures the environments scan of the same
// tree.
func BenchmarkLoadEnvironments(b *testing.B) {
	root := benchTree(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := LoadEnvironments(root); err != nil {
			b.Fatal(err)
		}
	}
}
