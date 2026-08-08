package render

import (
	"fmt"
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
