package codeeditor

import "testing"

func TestTruncateRunesAnsi(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"plain text", 5, "plai…"},
		{"plain", 10, "plain"},
		{"", 5, ""},
		{"x", 1, "…"},
		{"\x1b[31mred\x1b[0m", 3, "\x1b[31mred\x1b[0m"},
		{"\x1b[31mred\x1b[0m", 9, "\x1b[31mred\x1b[0m"},
		{"\x1b[31mred\x1b[0m", 2, "\x1b[31mr…\x1b[0m"},
		{"\x1b[38;2;1;2;3mA\x1b[0m", 1, "…"},
		{"\x1b[38;2;1;2;3mAB\x1b[0m", 2, "\x1b[38;2;1;2;3mAB\x1b[0m"},
		{"\x1b[38;2;1;2;3mABC\x1b[0m", 2, "\x1b[38;2;1;2;3mA…\x1b[0m"},
	}
	for _, c := range cases {
		if got := TruncateRunesAnsi(c.in, c.n); got != c.want {
			t.Errorf("TruncateRunesAnsi(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}
