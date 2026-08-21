package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Dread-Code/lazypost/internal/collection"
)

// pattern matches {{name}} with optional inner whitespace, e.g. {{ host }}.
var pattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.\-]+)\s*\}\}`)

// Apply substitutes {{name}} placeholders in s using vars. Unknown
// placeholders are left untouched.
func Apply(s string, vars map[string]string) string {
	if len(vars) == 0 || s == "" {
		return s
	}
	return pattern.ReplaceAllStringFunc(s, func(m string) string {
		name := pattern.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
}

// Unresolved returns the names of any {{placeholders}} still present in s.
func Unresolved(s string) []string {
	ms := pattern.FindAllStringSubmatch(s, -1)
	names := make([]string, 0, len(ms))
	for _, m := range ms {
		names = append(names, m[1])
	}
	return names
}

// placeholderSentinel is a JSON number so unlikely to occur in real bodies
// that masking is collision-free in practice (checked anyway before use).
const placeholderSentinel = "1766847064778384329583297500742918515827483896875618958121606201292619776"

// FormatJSON pretty-prints s with 2-space indent. Valid JSON formats
// directly; bodies invalid only because of raw {{placeholders}} in value
// positions (e.g. `"userId": {{user_id}}`) are formatted by masking each
// placeholder with a unique sentinel, indenting, and restoring — so the
// structure formats while the placeholders survive verbatim. Anything
// else is returned untouched, idempotently.
func FormatJSON(s string) string {
	if json.Valid([]byte(s)) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(s), "", "  "); err == nil {
			return buf.String()
		}
		return s
	}
	if !strings.Contains(s, "{{") {
		return s
	}
	base := placeholderSentinel
	for strings.Contains(s, base) {
		base += "1" // pathological collision: shift the sentinel
	}
	var phs []string
	masked := pattern.ReplaceAllStringFunc(s, func(m string) string {
		sent := fmt.Sprintf("%s%04d", base, len(phs))
		phs = append(phs, m)
		return sent
	})
	if !json.Valid([]byte(masked)) {
		return s
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(masked), "", "  "); err != nil {
		return s
	}
	out := buf.String()
	for i, ph := range phs {
		out = strings.Replace(out, fmt.Sprintf("%s%04d", base, i), ph, 1)
	}
	return out
}

// Request returns a copy of req with all placeholders resolved. The
// original is never mutated; only the fields it owns are interpolated.
func Request(req collection.Request, vars map[string]string) collection.Request {
	out := req
	out.URL = Apply(req.URL, vars)
	out.Body = Apply(req.Body, vars)
	if len(req.Query) > 0 {
		out.Query = make([]collection.Param, len(req.Query))
		for i, p := range req.Query {
			out.Query[i] = collection.Param{Name: Apply(p.Name, vars), Value: Apply(p.Value, vars)}
		}
	}
	if len(req.Headers) > 0 {
		out.Headers = make([]collection.Header, len(req.Headers))
		for i, h := range req.Headers {
			out.Headers[i] = collection.Header{
				Name:  h.Name,
				Value: Apply(h.Value, vars),
			}
		}
	}
	if req.Auth != nil {
		// copy the Auth value so the original struct stays untouched
		a := *req.Auth
		a.Username = Apply(a.Username, vars)
		a.Password = Apply(a.Password, vars)
		a.Token = Apply(a.Token, vars)
		a.KeyName = Apply(a.KeyName, vars)
		a.KeyValue = Apply(a.KeyValue, vars)
		out.Auth = &a
	}
	return out
}
