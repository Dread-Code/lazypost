package importer

import (
	"net/url"
	"strings"

	"lazypost/internal/collection"
)

// normalizeURLQuery makes Request.Query the canonical representation for
// structured query parameters while preserving URL query parameters that the
// source did not model separately. URL fragments stay attached to the URL.
func normalizeURLQuery(rawURL string, explicit []collection.Param, explicitKeys map[string]struct{}) (string, []collection.Param) {
	base, raw := splitURLQuery(rawURL)
	if len(explicitKeys) == 0 {
		return base, append(raw, explicit...)
	}

	// Structured source fields are authoritative for keys they mention. This
	// also removes disabled source fields that are still present in a raw URL.
	merged := make([]collection.Param, 0, len(raw)+len(explicit))
	for _, param := range raw {
		if _, structured := explicitKeys[param.Name]; !structured {
			merged = append(merged, param)
		}
	}
	merged = append(merged, explicit...)
	return base, merged
}

func splitURLQuery(rawURL string) (string, []collection.Param) {
	fragment := ""
	if i := strings.IndexByte(rawURL, '#'); i >= 0 {
		fragment = rawURL[i:]
		rawURL = rawURL[:i]
	}
	queryIndex := strings.IndexByte(rawURL, '?')
	if queryIndex < 0 {
		return rawURL + fragment, nil
	}

	base := rawURL[:queryIndex] + fragment
	rawQuery := rawURL[queryIndex+1:]
	if rawQuery == "" {
		return base, nil
	}
	parts := strings.Split(rawQuery, "&")
	params := make([]collection.Param, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		name, value, _ := strings.Cut(part, "=")
		params = append(params, collection.Param{
			Name:  unescapeQueryPart(name),
			Value: unescapeQueryPart(value),
		})
	}
	return base, params
}

func unescapeQueryPart(value string) string {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}
