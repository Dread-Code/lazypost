package curl

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Dread-Code/lazypost/internal/collection"
)

var noArgFlags = map[string]bool{
	"-s": true, "--silent": true,
	"-S": true, "--show-error": true,
	"-k": true, "--insecure": true,
	"-i": true, "--include": true,
	"-v": true, "--verbose": true,
	"-L": true, "--location": true,
	"-f": true, "--fail": true,
	"-g": true, "--globoff": true,
	"-G": true, "--get": true,
	"-N": true, "--no-buffer": true,
	"-q":           true,
	"--compressed": true, "--no-progress-meter": true,
	"--http1.1": true, "--http2": true,
}

// valueFlags are request-shaping flags lazypost does not model; their value
// is consumed and skipped so parsing can continue.
var valueFlags = map[string]bool{
	"-o": true, "--output": true,
	"-A": true, "--user-agent": true,
	"-e": true, "--referer": true,
	"-b": true, "--cookie": true,
	"-c": true, "--cookie-jar": true,
	"-x": true, "--proxy": true,
	"-U": true, "--proxy-user": true,
	"-m": true, "--max-time": true,
	"--connect-timeout": true,
	"-w":                true, "--write-out": true,
	"-D": true, "--dump-header": true,
	"-T": true, "--upload-file": true,
	"-r": true, "--range": true,
	"-K": true, "--config": true,
	"--resolve": true, "--retry": true,
	"--cacert": true, "--cert": true, "--key": true,
}

const noArgShorts = "sSkivLfGgGNq"

// Parse turns a curl command line into a collection.Request. Warnings about
// ignored request-affecting flags are intentionally discarded for backwards
// compatibility; callers that want them should use ParseWithWarnings.
func Parse(cmdline string) (*collection.Request, error) {
	req, _, err := ParseWithWarnings(cmdline)
	return req, err
}

// ParseWithWarnings parses a curl command and reports flags that lazypost
// cannot represent in collection.Request without changing request behavior.
func ParseWithWarnings(cmdline string) (*collection.Request, []string, error) {
	toks, err := tokenize(cmdline)
	if err != nil {
		return nil, nil, err
	}
	if len(toks) == 0 {
		return nil, nil, errors.New("empty command")
	}
	if base := toks[0]; base != "curl" && !strings.HasSuffix(base, "/curl") {
		return nil, nil, fmt.Errorf("not a curl command (starts with %q)", base)
	}

	req := &collection.Request{Method: "GET"}
	var body []string
	var warnings []string
	methodSet := false      // did the user pass -X explicitly?
	hasContentType := false // did any -H set Content-Type?
	getData := false        // did the user pass -G/--get?

	// nextArg returns the token after the current one, advancing the
	// loop index so the value isn't mistaken for a flag.
	nextArg := func(i *int, flag string) (string, error) {
		if *i+1 >= len(toks) {
			return "", fmt.Errorf("flag %s needs a value", flag)
		}
		*i++
		return toks[*i], nil
	}

	for i := 1; i < len(toks); i++ {
		t := toks[i]
		switch {
		case t == "-X" || t == "--request":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			req.Method = strings.ToUpper(v)
			methodSet = true

		case t == "-H" || t == "--header":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			name, value, ok := strings.Cut(v, ":")
			if !ok {
				return nil, nil, fmt.Errorf("malformed header %q", v)
			}
			name = strings.TrimSpace(name)
			value = strings.TrimSpace(value)
			if strings.EqualFold(name, "content-type") {
				hasContentType = true
			}
			req.Headers = append(req.Headers, collection.Header{Name: name, Value: value})

		case t == "-d" || t == "--data" || t == "--data-raw" || t == "--data-binary" || t == "--data-ascii":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			// curl joins repeated -d payloads with '&'
			body = append(body, v)

		case t == "--data-urlencode":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			body = append(body, urlEncodeData(v))

		case t == "-u" || t == "--user":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			user, pass, _ := strings.Cut(v, ":")
			req.Auth = &collection.Auth{Type: "basic", Username: user, Password: pass}

		case t == "--url":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, nil, err
			}
			req.URL = v

		case t == "-G" || t == "--get":
			getData = true

		case noArgFlags[t]:
			if warning, ok := ignoredFlagWarnings[t]; ok {
				warnings = append(warnings, warning)
			}

		case strings.HasPrefix(t, "--") && valueFlags[t]:
			// long value-taking flag: consume and drop its value
			if _, err := nextArg(&i, t); err != nil {
				return nil, nil, err
			}
			if warning, ok := ignoredFlagWarnings[t]; ok {
				warnings = append(warnings, warning)
			}

		case isShortFlagCluster(t):
			// e.g. -sSL: fine only when every flag in the cluster is ignorable

		case valueFlags[t]:
			// short value-taking flag: consume and drop its value
			if _, err := nextArg(&i, t); err != nil {
				return nil, nil, err
			}
			if warning, ok := ignoredFlagWarnings[t]; ok {
				warnings = append(warnings, warning)
			}

		case strings.HasPrefix(t, "-") && t != "-":
			return nil, nil, fmt.Errorf("unsupported flag %s", t)

		default:
			if req.URL != "" {
				return nil, nil, fmt.Errorf("unexpected argument %q", t)
			}
			req.URL = t
		}
	}

	if req.URL == "" {
		return nil, nil, errors.New("no URL in curl command")
	}
	if len(body) > 0 {
		if getData {
			for _, data := range body {
				req.Query = append(req.Query, parseGetData(data)...)
			}
		} else {
			req.Body = strings.Join(body, "&")
			if !methodSet {
				// curl's default: a body without -X becomes a POST
				req.Method = "POST"
			}
			if !hasContentType {
				// mirror curl's implicit form content-type
				req.Headers = append(req.Headers, collection.Header{
					Name:  "Content-Type",
					Value: "application/x-www-form-urlencoded",
				})
			}
		}
	}
	return req, warnings, nil
}

var ignoredFlagWarnings = map[string]string{
	"-k":                "TLS verification flag ignored during import",
	"--insecure":        "TLS verification flag ignored during import",
	"-A":                "user-agent flag ignored during import",
	"--user-agent":      "user-agent flag ignored during import",
	"-b":                "cookie flag ignored during import",
	"--cookie":          "cookie flag ignored during import",
	"-c":                "cookie-jar flag ignored during import",
	"--cookie-jar":      "cookie-jar flag ignored during import",
	"-x":                "proxy flag ignored during import",
	"--proxy":           "proxy flag ignored during import",
	"-U":                "proxy-user flag ignored during import",
	"--proxy-user":      "proxy-user flag ignored during import",
	"--resolve":         "resolve flag ignored during import",
	"--cacert":          "CA certificate flag ignored during import",
	"--cert":            "client certificate flag ignored during import",
	"--key":             "client key flag ignored during import",
	"-T":                "upload-file flag ignored during import",
	"--upload-file":     "upload-file flag ignored during import",
	"-o":                "output file flag ignored during import",
	"--output":          "output file flag ignored during import",
	"-m":                "max-time flag ignored during import",
	"--max-time":        "max-time flag ignored during import",
	"--connect-timeout": "connect-timeout flag ignored during import",
}

func parseGetData(data string) []collection.Param {
	var params []collection.Param
	for _, part := range strings.Split(data, "&") {
		if part == "" {
			continue
		}
		name, value, hasValue := strings.Cut(part, "=")
		if !hasValue {
			params = append(params, collection.Param{Name: queryUnescape(name)})
			continue
		}
		params = append(params, collection.Param{Name: queryUnescape(name), Value: queryUnescape(value)})
	}
	return params
}

func queryUnescape(value string) string {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

// isShortFlagCluster reports whether t is a short-flag bundle (like -sSL)
// in which every character is a harmless no-arg flag.
func isShortFlagCluster(t string) bool {
	if len(t) < 2 || t[0] != '-' || t[1] == '-' {
		return false
	}
	for _, r := range t[1:] {
		if !strings.ContainsRune(noArgShorts, r) {
			return false
		}
	}
	return true
}

// urlEncodeData mirrors curl --data-urlencode for "name=content" and bare
// "content" forms (file forms with @ are not supported).
func urlEncodeData(v string) string {
	name, content, ok := strings.Cut(v, "=")
	if !ok {
		return url.QueryEscape(v)
	}
	return name + "=" + url.QueryEscape(content)
}

// tokenize splits a command line into arguments, honoring single quotes,
// double quotes, backslash escapes, and backslash-newline continuations.
func tokenize(s string) ([]string, error) {
	var toks []string
	var cur strings.Builder
	inTok := false
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '\\' && i+1 < len(s) && s[i+1] == '\n':
			i += 2 // line continuation
		case c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
			inTok = true
			i++
		case c == '\'':
			j := strings.IndexByte(s[i+1:], '\'')
			if j < 0 {
				return nil, errors.New("unclosed single quote")
			}
			cur.WriteString(s[i+1 : i+1+j])
			inTok = true
			i += j + 2
		case c == '"':
			i++
			closed := false
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					cur.WriteByte(s[i+1])
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					closed = true
					break
				}
				cur.WriteByte(s[i])
				i++
			}
			if !closed {
				return nil, errors.New("unclosed double quote")
			}
			inTok = true
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if inTok {
				toks = append(toks, cur.String())
				cur.Reset()
				inTok = false
			}
			i++
		default:
			cur.WriteByte(c)
			inTok = true
			i++
		}
	}
	if inTok {
		toks = append(toks, cur.String())
	}
	return toks, nil
}

// Format renders req as a curl one-liner. Known {{vars}} are expected to
// have been interpolated by the caller; any that remain are kept raw so
// the receiver notices unresolved variables.
func Format(req collection.Request) string {
	parts := []string{"curl"}
	method := strings.ToUpper(req.Method)
	if method != "" && method != "GET" {
		parts = append(parts, "-X", method)
	}

	urlStr := req.URL
	if parsed, err := url.Parse(req.URL); err == nil {
		values := parsed.Query()
		for _, p := range req.Query {
			values.Add(p.Name, p.Value)
		}
		if req.Auth != nil && req.Auth.Type == "apikey" && strings.EqualFold(req.Auth.KeyIn, "query") {
			values.Set(req.Auth.KeyName, req.Auth.KeyValue)
		}
		parsed.RawQuery = values.Encode()
		urlStr = restorePlaceholders(parsed.String())
	} else if req.Auth != nil && req.Auth.Type == "apikey" && strings.EqualFold(req.Auth.KeyIn, "query") {
		// Keep formatting useful for an invalid/unresolved URL while still
		// escaping the API key value.
		urlStr += "?" + url.QueryEscape(req.Auth.KeyName) + "=" + url.QueryEscape(req.Auth.KeyValue)
	}
	parts = append(parts, shquote(urlStr))

	for _, h := range req.Headers {
		parts = append(parts, "-H", shquote(h.Name+": "+h.Value))
	}
	if req.Auth != nil {
		switch req.Auth.Type {
		case "basic":
			parts = append(parts, "-u", shquote(req.Auth.Username+":"+req.Auth.Password))
		case "bearer":
			parts = append(parts, "-H", shquote("Authorization: Bearer "+req.Auth.Token))
		case "apikey":
			if !strings.EqualFold(req.Auth.KeyIn, "query") {
				parts = append(parts, "-H", shquote(req.Auth.KeyName+": "+req.Auth.KeyValue))
			}
		}
	}
	if req.Body != "" {
		parts = append(parts, "--data-raw", shquote(req.Body))
	}
	return strings.Join(parts, " ")
}

func restorePlaceholders(value string) string {
	return strings.NewReplacer(
		"%7B%7B", "{{",
		"%7D%7D", "}}",
		"%7b%7b", "{{",
		"%7d%7d", "}}",
	).Replace(value)
}

// shquote wraps s in single quotes, escaping embedded single quotes the
// shell way ('\”), so the output is safe to paste into a shell.
func shquote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
