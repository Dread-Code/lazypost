package collection

import (
	"testing"
)

func TestLoadSampleCollections(t *testing.T) {
	entries, err := Load("../../sample-collections")
	if err != nil {
		t.Fatalf("Load sample-collections: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("sample-collections is empty")
	}
	reqs := 0
	for _, e := range entries {
		if e.Kind != Req {
			continue
		}
		reqs++
		if e.Req.URL == "" {
			t.Errorf("request %q has empty URL", e.Name)
		}
		if e.Req.Method == "" {
			t.Errorf("request %q has empty method", e.Name)
		}
	}
	if reqs < 5 {
		t.Errorf("expected at least 5 sample requests, got %d", reqs)
	}

	envs, names, err := LoadEnvironments("../../sample-collections")
	if err != nil {
		t.Fatalf("LoadEnvironments: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected dev+prod environments, got %v", names)
	}
	if envs["dev"]["host"] != "https://zenquotes.io" {
		t.Errorf("dev host = %q", envs["dev"]["host"])
	}
}
