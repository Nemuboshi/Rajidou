package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(p, []byte("links: ["), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected yaml error")
	}
}

func TestLoadEmptyLinks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(p, []byte("links: []\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load(p)
	if err == nil {
		t.Fatal("expected empty links error")
	}
}

func TestLoadCustomJobs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ok.yaml")
	if err := os.WriteFile(p, []byte("links:\n  - https://example.com\njobs: 7\noutputDir: out\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.Jobs != 7 || c.OutputDir != "out" {
		t.Fatalf("unexpected config: %+v", c)
	}
}
