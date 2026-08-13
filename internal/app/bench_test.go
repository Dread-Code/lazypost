package app

import (
	"testing"

	"lazypost/internal/collection"
	"lazypost/internal/httpclient"
)

// BenchmarkSend measures the send pipeline with hooks, interpolation,
// and the chain store behind a fake client — the per-send cost.
func BenchmarkSend(b *testing.B) {
	req := collection.Request{
		Method: "POST",
		URL:    "{{host}}/posts",
		Headers: []collection.Header{
			{Name: "Content-Type", Value: "application/json"},
		},
		Body: `{"title": "{{title}}", "userId": 1}`,
		Pre:  `req.headers["X-Client"] = "lazypost"`,
		Post: `return response.status_code == 200`,
	}
	vars := map[string]string{"host": "https://api.test", "title": "hello"}
	store := map[string]string{"user_id": "7"}
	client := Client(func(req collection.Request, v map[string]string) (*httpclient.Response, error) {
		return fakeResponse(), nil
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Send(client, req, vars, store); err != nil {
			b.Fatal(err)
		}
	}
}
