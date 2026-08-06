package render

import (
	"regexp"

	"postgo/internal/collection"
)

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

// Request returns a copy of req with all placeholders resolved.
func Request(req collection.Request, vars map[string]string) collection.Request {
	out := req
	out.URL = Apply(req.URL, vars)
	out.Body = Apply(req.Body, vars)
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
