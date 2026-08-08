package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"lazypost/internal/collection"
	"lazypost/internal/render"
)

type Response struct {
	StatusCode  int
	Status      string
	Headers     http.Header
	Body        []byte
	Duration    time.Duration
	ContentType string
}

var client = &http.Client{Timeout: 30 * time.Second}

// Exec builds and executes the HTTP request described by req after
// interpolating vars, and returns the raw response. Transport errors
// (dial/TLS/timeout) are errors; HTTP error statuses are not.
func Exec(req collection.Request, vars map[string]string) (*Response, error) {
	req = render.Request(req, vars)

	// a leftover placeholder in the URL can never succeed — fail loudly
	// with an actionable message instead of a cryptic transport error
	if missing := render.Unresolved(req.URL); len(missing) > 0 {
		return nil, fmt.Errorf("unresolved placeholder {{%s}} in URL — activate an environment with ctrl+e",
			strings.Join(missing, "}}, {{"))
	}

	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequest(method, req.URL, body)
	if err != nil {
		return nil, err
	}

	// Merge query params: URL-string params first, then explicit params,
	// then apikey-in-query which overrides any colliding key.
	q := httpReq.URL.Query()
	for _, p := range req.Query {
		q.Add(p.Name, p.Value)
	}
	if req.Auth != nil && strings.EqualFold(req.Auth.Type, "apikey") && strings.EqualFold(req.Auth.KeyIn, "query") {
		q.Set(req.Auth.KeyName, req.Auth.KeyValue)
	}
	httpReq.URL.RawQuery = q.Encode()

	// Header.Add keeps duplicate header names rather than replacing
	for _, h := range req.Headers {
		httpReq.Header.Add(h.Name, h.Value)
	}
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			httpReq.SetBasicAuth(req.Auth.Username, req.Auth.Password)
		case "bearer":
			httpReq.Header.Set("Authorization", "Bearer "+req.Auth.Token)
		case "apikey":
			if !strings.EqualFold(req.Auth.KeyIn, "query") {
				httpReq.Header.Set(req.Auth.KeyName, req.Auth.KeyValue)
			}
		}
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode:  resp.StatusCode,
		Status:      resp.Status,
		Headers:     resp.Header,
		Body:        raw,
		Duration:    time.Since(start),
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// FormattedBody returns the body pretty-printed when it is valid JSON,
// otherwise the raw body.
func (r *Response) FormattedBody() string {
	if json.Valid(r.Body) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, r.Body, "", "  "); err == nil {
			return buf.String()
		}
	}
	return string(r.Body)
}

// FormattedHeaders returns response headers as sorted "Name: Value" lines.
func (r *Response) FormattedHeaders() string {
	names := make([]string, 0, len(r.Headers))
	for name := range r.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		for _, v := range r.Headers[name] {
			fmt.Fprintf(&b, "%s: %s\n", name, v)
		}
	}
	return b.String()
}

// Summary renders the one-line "Status · size · duration" for the pane
// title, colored by status class in the response pane.
func (r *Response) Summary() string {
	return fmt.Sprintf("%s · %s · %s",
		r.Status,
		humanSize(len(r.Body)),
		r.Duration.Round(time.Millisecond),
	)
}

// humanSize renders n bytes as e.g. "1.2 KiB".
func humanSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	// walk div up to n's unit while exp counts the suffix index
	div, exp := unit, 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
