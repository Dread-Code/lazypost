package curl

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"postgo/internal/collection"
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

// valueFlags are request-shaping flags postgo does not model; their value
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

// Parse turns a curl command line into a collection.Request. It mirrors
// curl's semantics for the common flags; unsupported flags are errors.
func Parse(cmdline string) (*collection.Request, error) {
	toks, err := tokenize(cmdline)
	if err != nil {
		return nil, err
	}
	if len(toks) == 0 {
		return nil, errors.New("empty command")
	}
	if base := toks[0]; base != "curl" && !strings.HasSuffix(base, "/curl") {
		return nil, fmt.Errorf("not a curl command (starts with %q)", base)
	}

	req := &collection.Request{Method: "GET"}
	var body []string
	methodSet := false
	hasContentType := false

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
				return nil, err
			}
			req.Method = strings.ToUpper(v)
			methodSet = true

		case t == "-H" || t == "--header":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, err
			}
			name, value, ok := strings.Cut(v, ":")
			if !ok {
				return nil, fmt.Errorf("malformed header %q", v)
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
				return nil, err
			}
			body = append(body, v)

		case t == "--data-urlencode":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, err
			}
			body = append(body, urlEncodeData(v))

		case t == "-u" || t == "--user":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, err
			}
			user, pass, _ := strings.Cut(v, ":")
			req.Auth = &collection.Auth{Type: "basic", Username: user, Password: pass}

		case t == "--url":
			v, err := nextArg(&i, t)
			if err != nil {
				return nil, err
			}
			req.URL = v

		case noArgFlags[t]:
			// harmless for request building; skip

		case strings.HasPrefix(t, "--") && valueFlags[t]:
			if _, err := nextArg(&i, t); err != nil {
				return nil, err
			}

		case isShortFlagCluster(t):
			// e.g. -sSL: fine only when every flag in the cluster is ignorable

		case valueFlags[t]:
			if _, err := nextArg(&i, t); err != nil {
				return nil, err
			}

		case strings.HasPrefix(t, "-") && t != "-":
			return nil, fmt.Errorf("unsupported flag %s", t)

		default:
			if req.URL != "" {
				return nil, fmt.Errorf("unexpected argument %q", t)
			}
			req.URL = t
		}
	}

	if req.URL == "" {
		return nil, errors.New("no URL in curl command")
	}
	if len(body) > 0 {
		req.Body = strings.Join(body, "&")
		if !methodSet {
			req.Method = "POST"
		}
		if !hasContentType {
			req.Headers = append(req.Headers, collection.Header{
				Name:  "Content-Type",
				Value: "application/x-www-form-urlencoded",
			})
		}
	}
	return req, nil
}

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

// Format renders req as a curl one-liner. Placeholders like {{host}} are
// kept raw so the receiver notices unresolved variables.
func Format(req collection.Request) string {
	parts := []string{"curl"}
	method := strings.ToUpper(req.Method)
	if method != "" && method != "GET" {
		parts = append(parts, "-X", method)
	}

	urlStr := req.URL
	if req.Auth != nil && req.Auth.Type == "apikey" && strings.EqualFold(req.Auth.KeyIn, "query") {
		sep := "?"
		if strings.Contains(urlStr, "?") {
			sep = "&"
		}
		urlStr += sep + req.Auth.KeyName + "=" + req.Auth.KeyValue
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

func shquote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
