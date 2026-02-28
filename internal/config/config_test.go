package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultJobs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte("links:\n  - https://example.com\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Jobs != 2 {
		t.Fatalf("want jobs=2, got %d", c.Jobs)
	}
}
