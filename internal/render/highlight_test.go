package render

import (
	"fmt"
	"strings"
	"testing"
)

// paint marks each token with its kind so tests can assert the exact
// token stream, not the colors.
func paint(k Kind, lit string) string { return fmt.Sprintf("<%d>%s</%d>", k, lit, k) }

// helpers compose expectations from the Kind constants, no magic numbers.
func str(lit string) string { return fmt.Sprintf("<%d>%s</%d>", KindString, lit, KindString) }
func num(lit string) string { return fmt.Sprintf("<%d>%s</%d>", KindNumber, lit, KindNumber) }
func lit(lit string) string { return fmt.Sprintf("<%d>%s</%d>", KindLiteral, lit, KindLiteral) }
func key(lit string) string { return fmt.Sprintf("<%d>%s</%d>", KindKey, lit, KindKey) }
func pn(lit string) string  { return fmt.Sprintf("<%d>%s</%d>", KindPunctuation, lit, KindPunctuation) }

func TestHighlightJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"object with every kind",
			`{"name":"post","count":2,"ok":true,"none":null,"n":-1.5e-3}`,
			pn("{") + key(`"name"`) + pn(":") + str(`"post"`) + pn(",") +
				key(`"count"`) + pn(":") + num("2") + pn(",") +
				key(`"ok"`) + pn(":") + lit("true") + pn(",") +
				key(`"none"`) + pn(":") + lit("null") + pn(",") +
				key(`"n"`) + pn(":") + num("-1.5e-3") + pn("}"),
		},
		{
			"keys vs value strings nested",
			`{"a":[1,"two"],"b":{"c":"x"}}`,
			pn("{") + key(`"a"`) + pn(":[") + num("1") + pn(",") +
				str(`"two"`) + pn("],") +
				key(`"b"`) + pn(":{") + key(`"c"`) + pn(":") +
				str(`"x"`) + pn("}}"),
		},
		{
			"escaped quotes stay inside the string",
			`{"q":"say \"hi\" and \\ x"}`,
			pn("{") + key(`"q"`) + pn(":") + str(`"say \"hi\" and \\ x"`) + pn("}"),
		},
		{
			"unicode in keys and values",
			`{"м":"🎉"}`,
			pn("{") + key(`"м"`) + pn(":") + str(`"🎉"`) + pn("}"),
		},
		{
			"number forms",
			`[0,-0.5,1E+10,2.5e-3]`,
			pn("[") + num("0") + pn(",") + num("-0.5") + pn(",") +
				num("1E+10") + pn(",") + num("2.5e-3") + pn("]"),
		},
		{
			"empty containers",
			`{"e":{},"a":[],"s":""}`,
			pn("{") + key(`"e"`) + pn(":{},") +
				key(`"a"`) + pn(":[],") +
				key(`"s"`) + pn(":") + str(`""`) + pn("}"),
		},
		{
			"whitespace preserved",
			"{\n  \"a\": 1\n}",
			pn("{") + "\n  " + key(`"a"`) + pn(":") + " " + num("1") + "\n" + pn("}"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HighlightJSON(c.in, paint); got != c.want {
				t.Errorf("HighlightJSON(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestHighlightJSONInvalidIsIdentity(t *testing.T) {
	cases := []string{
		"",
		"not json",
		`{"a":`,
		`"unterminated`,
		"123 456",
		`{"a":1} trailing`,
	}
	for _, in := range cases {
		if got := HighlightJSON(in, paint); got != in {
			t.Errorf("HighlightJSON(%q) = %q, want identity", in, got)
		}
	}
}

// The editor lexes the whole buffer at once, so object context survives
// across lines: a key on a line after the opening brace must stay a key.
// A line-scoped lex reclassifies every quoted string as a string and the
// body renders in one color — the reported bug.
func TestHighlightJSONLinesKeysStayKeys(t *testing.T) {
	body := `{
  "title": "{{title}}",
  "userId": 42
}`
	want := []string{
		pn("{"),
		"  " + key(`"title"`) + pn(":") + " " + str(`"{{title}}"`) + pn(","),
		"  " + key(`"userId"`) + pn(":") + " " + num("42"),
		pn("}"),
	}
	got := HighlightJSONLines(body, paint)
	if len(got) != len(want) {
		t.Fatalf("HighlightJSONLines returned %d lines, want %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want[i])
		}
	}
	// a string on the first line lexes as a string, not a key: the
	// context, not the line position, decides
	got = HighlightJSONLines(`"plain"`, paint)
	if len(got) != 1 || got[0] != str(`"plain"`) {
		t.Errorf("first-line string = %q, want %q", got, str(`"plain"`))
	}
}

// HighlightJSONLines has no validity gate: it colors the buffer while it
// is half-typed, and it must never lose or invent text (joining the
// lines and stripping the markers yields the input).
func TestHighlightJSONLinesFragmentWhileInvalid(t *testing.T) {
	got := HighlightJSONLines(`{"title": "`, paint)
	if len(got) != 1 || !strings.Contains(got[0], "<") {
		t.Errorf("fragment should color tokens of invalid JSON, got %q", got)
	}
	cases := []string{
		`{"title": "`,
		"{\n  \"a\": 1,\n",
		"not json",
		"",
	}
	for _, in := range cases {
		got := HighlightJSONLines(in, paint)
		if joined := stripMarkers(strings.Join(got, "\n")); joined != in {
			t.Errorf("reconstruction failed for %q: got %q", in, joined)
		}
	}
}

// TestHighlightJSONSplit pins the cursor-seam contract: the prefix
// restores the lexer state (so `"title"` after `{` is a key, not a
// string), and a cut inside a token keeps the color on both sides.
func TestHighlightJSONSplit(t *testing.T) {
	// the prefix matters: without it the line lexes from root and the
	// key becomes a string
	_, without := HighlightJSONSplit("", `  "title": "hi"`, 0, paint)
	if !strings.Contains(without, str(`"title"`)) {
		t.Errorf("no-prefix line should lex the key as a string, got %q", without)
	}
	_, with := HighlightJSONSplit("{\n", `  "title": "hi"`, 0, paint)
	if !strings.Contains(with, key(`"title"`)) {
		t.Errorf("prefixed line should keep the key color, got %q", with)
	}
	// cut on a token boundary (the key's closing quote): both pieces
	// join to the full painted line
	pre, post := HighlightJSONSplit("{\n", `  "title": "hi"`, len(`  "title"`), paint)
	if pre != "  "+key(`"title"`) {
		t.Errorf("pre = %q, want %q", pre, "  "+key(`"title"`))
	}
	if post != pn(":")+" "+str(`"hi"`) {
		t.Errorf("post = %q, want %q", post, pn(":")+" "+str(`"hi"`))
	}
	// mid-token cut inside the key: both halves keep the key color
	pre, post = HighlightJSONSplit("{\n", `  "title": "hi"`, 7, paint)
	if pre != "  "+key(`"titl`) {
		t.Errorf("mid-key pre = %q, want %q", pre, "  "+key(`"titl`))
	}
	if post != key(`e"`)+pn(":")+" "+str(`"hi"`) {
		t.Errorf("mid-key post = %q", post)
	}
	// mid-token cut inside a value string: the color survives the seam
	pre, post = HighlightJSONSplit("{\n", `  "title": "hi"`, 12, paint)
	if pre != "  "+key(`"title"`)+pn(":")+" "+str(`"`) {
		t.Errorf("mid-string pre = %q", pre)
	}
	if post != str(`hi"`) {
		t.Errorf("mid-string post = %q, want %q", post, str(`hi"`))
	}
	// cuts are clamped to the line
	if pre, post := HighlightJSONSplit("{\n", `  "x": 1`, 999, paint); pre == "" || post != "" {
		t.Errorf("overlong cut: pre = %q, post = %q", pre, post)
	}
	if pre, post := HighlightJSONSplit("{\n", `  "x": 1`, -3, paint); pre != "" || post == "" {
		t.Errorf("negative cut: pre = %q, post = %q", pre, post)
	}
}
