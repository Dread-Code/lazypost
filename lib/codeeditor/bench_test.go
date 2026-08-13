package codeeditor

import (
	"fmt"
	"strings"
	"testing"
)

// bigBody returns a synthetic n-line buffer.
func bigBody(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, `{"key%04d": "value %d", "ok": %t}`+"\n", i, i, i%2 == 0)
	}
	return b.String()
}

// BenchmarkView measures the widget core per render: whole-buffer
// painting split into per-line pieces, gutter, and filler rows on a
// 500-line buffer (identity highlighter — the chroma cost lives in the
// lazypost-side BenchmarkRenderJSONBody).
func BenchmarkView(b *testing.B) {
	e := New(80, 40, "", markerHighlighter{})
	e.SetValue(bigBody(500))
	e.Focus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.View()
	}
}

// BenchmarkViewWithCursor measures the same render with the cursor line
// re-painted through the context-aware Split path.
func BenchmarkViewWithCursor(b *testing.B) {
	e := New(80, 40, "", markerHighlighter{})
	e.SetValue(bigBody(500))
	e.Focus()
	e.SetCursor(len([]rune(bigBody(250)))) // mid-buffer, cursor line active
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.View()
	}
}
