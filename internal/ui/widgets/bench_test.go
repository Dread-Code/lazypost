package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Dread-Code/codeeditor"
)

// BenchmarkRenderJSONBody measures the real chroma JSON pipeline per
// render — the per-frame cost of the body editor on a 500-line body.
func BenchmarkRenderJSONBody(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&sb, `  "key%04d": "value %d",`+"\n", i, i)
	}
	body := "{\n" + sb.String() + "}\n"

	e := codeeditor.New(80, 40, "", jsonHighlighter())
	e.SetValue(body)
	e.Focus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.View()
	}
}
