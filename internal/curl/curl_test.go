package curl

import (
	"strings"
	"testing"

	"lazypost/internal/collection"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		method  string
		url     string
		body    string
		authTyp string
		nHdrs   int
		wantErr string
	}{
		{
			name:   "simple get",
			in:     "curl https://api.test/users",
			method: "GET", url: "https://api.test/users",
		},
		{
			name:   "explicit method",
			in:     "curl -X DELETE https://api.test/users/1",
			method: "DELETE", url: "https://api.test/users/1",
		},
		{
			name:   "data implies post",
			in:     `curl -d 'title=hello' https://api.test/posts`,
			method: "POST", url: "https://api.test/posts",
			body: "title=hello", nHdrs: 1, // added Content-Type
		},
		{
			name:   "data with explicit get keeps get",
			in:     `curl -X GET -d 'q=1' https://api.test/search`,
			method: "GET", url: "https://api.test/search",
			body: "q=1", nHdrs: 1,
		},
		{
			name:   "headers and multiple data joined",
			in:     `curl -H 'Content-Type: application/json' -H 'X-A: 1' -d '{"a":1}' -d '{"b":2}' https://api.test/x`,
			method: "POST", url: "https://api.test/x",
			body: `{"a":1}&{"b":2}`, nHdrs: 2, // explicit Content-Type, no duplicate added
		},
		{
			name:   "basic auth",
			in:     `curl -u alice:secret https://api.test/x`,
			method: "GET", url: "https://api.test/x", authTyp: "basic",
		},
		{
			name:   "data-urlencode",
			in:     `curl --data-urlencode 'q=hello world' https://api.test/search`,
			method: "POST", url: "https://api.test/search",
			body: "q=hello+world", nHdrs: 1,
		},
		{
			name:   "ignored and skipped flags",
			in:     `curl -sSL -A 'bot/1.0' --connect-timeout 5 -k https://api.test/x`,
			method: "GET", url: "https://api.test/x",
		},
		{
			name:   "line continuations",
			in:     "curl -X POST \\\n  -H 'X-A: 1' \\\n  https://api.test/x",
			method: "POST", url: "https://api.test/x", nHdrs: 1,
		},
		{
			name:   "url flag",
			in:     `curl --url https://api.test/x`,
			method: "GET", url: "https://api.test/x",
		},
		{
			name:    "unsupported flag",
			in:      `curl -F 'file=@x' https://api.test/x`,
			wantErr: "unsupported flag -F",
		},
		{
			name:    "no url",
			in:      `curl -X GET`,
			wantErr: "no URL",
		},
		{
			name:    "unclosed quote",
			in:      `curl 'https://api.test/x`,
			wantErr: "unclosed",
		},
		{
			name:    "not curl",
			in:      `wget https://api.test/x`,
			wantErr: "not a curl command",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, err := Parse(c.in)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err = %v, want contains %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if req.Method != c.method {
				t.Errorf("method = %q, want %q", req.Method, c.method)
			}
			if req.URL != c.url {
				t.Errorf("url = %q, want %q", req.URL, c.url)
			}
			if req.Body != c.body {
				t.Errorf("body = %q, want %q", req.Body, c.body)
			}
			if len(req.Headers) != c.nHdrs {
				t.Errorf("headers = %+v, want %d", req.Headers, c.nHdrs)
			}
			if c.authTyp != "" {
				if req.Auth == nil || req.Auth.Type != c.authTyp {
					t.Errorf("auth = %+v, want type %q", req.Auth, c.authTyp)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string // substrings that must appear in order
	}{
		{
			name: "get stays implicit",
			in:   "curl https://api.test/users",
			want: []string{"curl 'https://api.test/users'"},
		},
		{
			name: "post with body and headers",
			in:   `curl -X POST -H 'Content-Type: application/json' -d '{"a":1}' https://api.test/x`,
			want: []string{"-X POST", "'https://api.test/x'", "-H 'Content-Type: application/json'", "--data-raw '{\"a\":1}'"},
		},
		{
			name: "basic auth",
			in:   `curl -u alice:secret https://api.test/x`,
			want: []string{"-u 'alice:secret'"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, err := Parse(c.in)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := Format(*req)
			last := -1
			for _, w := range c.want {
				idx := strings.Index(got, w)
				if idx < 0 || idx < last {
					t.Errorf("Format = %q, missing or out-of-order %q", got, w)
				}
				last = idx
			}
		})
	}
}

func TestFormatBearerAndAPIKey(t *testing.T) {
	parsed, err := Parse(`curl -H 'Authorization: Bearer tok' https://api.test/x`)
	if err != nil {
		t.Fatal(err)
	}
	if got := Format(*parsed); !strings.Contains(got, "-H 'Authorization: Bearer tok'") {
		t.Errorf("Format = %q", got)
	}
}

func TestFormatAppendsQueryParams(t *testing.T) {
	req, err := Parse(`curl 'https://api.test/search'`)
	if err != nil {
		t.Fatal(err)
	}
	req.Query = []collection.Param{{Name: "q", Value: "hello world"}, {Name: "tag", Value: "news"}}

	got := Format(*req)
	if !strings.Contains(got, "'https://api.test/search?q=hello+world&tag=news'") &&
		!strings.Contains(got, "'https://api.test/search?tag=news&q=hello+world'") {
		t.Errorf("Format = %q, want query params in URL", got)
	}
}

func TestRoundTrip(t *testing.T) {
	in := `curl -X PUT -H 'X-Token: abc' -d '{"v":2}' https://api.test/items/7`
	req, err := Parse(in)
	if err != nil {
		t.Fatal(err)
	}
	out := Format(*req)
	req2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-parse %q: %v", out, err)
	}
	if req2.Method != req.Method || req2.URL != req.URL || req2.Body != req.Body || len(req2.Headers) != len(req.Headers) {
		t.Errorf("round trip changed: %+v -> %+v", req, req2)
	}
}
