package render

import (
	"fmt"
	"strings"
	"testing"
)

func luaPaint(k LuaKind, lit string) string { return fmt.Sprintf("<%d>%s</%d>", k, lit, k) }

func kw(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaKeyword, lit, LuaKeyword) }
func st(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaString, lit, LuaString) }
func cm(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaComment, lit, LuaComment) }
func nu(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaNumber, lit, LuaNumber) }
func id(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaIdentifier, lit, LuaIdentifier) }
func op(lit string) string { return fmt.Sprintf("<%d>%s</%d>", LuaOperator, lit, LuaOperator) }

func TestHighlightLua(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"keywords strings numbers comments",
			`local name = "post"
-- a comment
if count >= 2 then return true end`,
			kw("local") + " " + id("name") + " " + op("=") + " " + st(`"post"`) + "\n" +
				cm("-- a comment") + "\n" +
				kw("if") + " " + id("count") + " " + op(">=") + " " + nu("2") + " " +
				kw("then") + " " + kw("return") + " " + kw("true") + " " + kw("end"),
		},
		{
			"single and double quoted strings with escapes",
			`local a = 'it\'s'
local b = "a \"b\""`,
			// chroma splits escape sequences into their own string tokens
			kw("local") + " " + id("a") + " " + op("=") + " " + st(`'it`) + st(`\'`) + st(`s'`) + "\n" +
				kw("local") + " " + id("b") + " " + op("=") + " " + st(`"a `) + st(`\"`) + st(`b`) + st(`\"`) + st(`"`),
		},
		{
			"block comment",
			`--[[ block
comment ]]`,
			cm("--[[ block\ncomment ]]"),
		},
		{
			"long string with level",
			"local doc = [=[lvl\n2]=]",
			kw("local") + " " + id("doc") + " " + op("=") + " " + st("[=[lvl\n2]=]"),
		},
		{
			"number forms",
			`[1, 2.5, -0.5e-3, 0x1F, 1e10]`,
			// a leading minus is its own operator token
			op("[") + nu("1") + op(",") + " " + nu("2.5") + op(",") + " " + op("-") +
				nu("0.5e-3") + op(",") + " " + nu("0x1F") + op(",") + " " + nu("1e10") + op("]"),
		},
		{
			"operators",
			`local t = a..b or not c`,
			// and/or/not are OperatorWord tokens, not keywords
			kw("local") + " " + id("t") + " " + op("=") + " " + id("a") + op("..") +
				id("b") + " " + op("or") + " " + op("not") + " " + id("c"),
		},
		{
			"empty",
			"",
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HighlightLua(c.in, luaPaint); got != c.want {
				t.Errorf("HighlightLua(%q) =\n%q\nwant\n%q", c.in, got, c.want)
			}
		})
	}
}

// TestHighlightLuaReconstructs pins the invariant that highlighting never
// loses or invents text: stripping the markers must yield the input.
func TestHighlightLuaReconstructs(t *testing.T) {
	cases := []string{
		"local x = 1",
		"--[[ unterminated block comment",
		`local s = "unterminated string`,
		"a = [=[unterminated long",
		"return #@%{}",
		"local  = weird",
	}
	for _, in := range cases {
		got := HighlightLua(in, luaPaint)
		if !strings.Contains(got, "<") || got == in {
			continue
		}
		got = stripMarkers(got)
		if got != in {
			t.Errorf("reconstruction failed for %q: got %q", in, got)
		}
	}
	// unterminated input must never panic or fail
	for _, in := range cases {
		if got := HighlightLua(in, luaPaint); !strings.Contains(got, "<") && got != in {
			t.Errorf("unterminated %q changed text: %q", in, got)
		}
	}
}

// TestHighlightLuaLines: whole-buffer lexing colors multi-line strings
// and block comments on every continuation line, and joining the lines
// reconstructs the input exactly.
func TestHighlightLuaLines(t *testing.T) {
	got := HighlightLuaLines("--[[ block\ncomment ]]", luaPaint)
	want := []string{cm("--[[ block"), cm("comment ]]")}
	if len(got) != len(want) {
		t.Fatalf("HighlightLuaLines returned %d lines, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want[i])
		}
	}
	// a long string keeps its color on the continuation line
	got = HighlightLuaLines("local doc = [=[lvl\n2]=]", luaPaint)
	if len(got) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(got))
	}
	if !strings.Contains(got[1], st(`2]=]`)) {
		t.Errorf("continuation line not colored as a string, got %q", got[1])
	}
	// reconstruction invariant: markers stripped, lines joined == input
	for _, in := range []string{
		"local x = 1",
		"--[[ unterminated block comment",
		"local a = \"multi\nline\"",
		"",
	} {
		got := HighlightLuaLines(in, luaPaint)
		if joined := stripMarkers(strings.Join(got, "\n")); joined != in {
			t.Errorf("reconstruction failed for %q: got %q", in, joined)
		}
	}
}

// TestHighlightLuaSplit: the cursor seam keeps colors on both sides of a
// cut inside a multi-line string.
func TestHighlightLuaSplit(t *testing.T) {
	prefix := "local s = [=[lvl\n"
	line := "2]=]"
	// cut inside the continuation: both halves stay string-colored
	pre, post := HighlightLuaSplit(prefix, line, 1, luaPaint)
	if pre != st("2") {
		t.Errorf("pre = %q, want %q", pre, st("2"))
	}
	if post != st("]=]") {
		t.Errorf("post = %q, want %q", post, st("]=]"))
	}
}

func stripMarkers(s string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, "<")
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		j := strings.Index(s[i:], ">")
		if j < 0 {
			b.WriteString(s[i:])
			return b.String()
		}
		s = s[i+j+1:]
		if k := strings.Index(s, "</"); k >= 0 {
			if e := strings.Index(s[k:], ">"); e >= 0 {
				b.WriteString(s[:k])
				s = s[k+e+1:]
				continue
			}
		}
	}
}
