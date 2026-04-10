package config

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/fsys"
)

func TestReadPacksLock_MissingFile(t *testing.T) {
	fs := fsys.NewFake()

	entries, err := ReadPacksLock(fs, "/city")
	if err != nil {
		t.Fatalf("ReadPacksLock: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestReadPacksLock_ParsesEntries(t *testing.T) {
	fs := fsys.NewFake()
	fs.Files["/city/packs.lock"] = []byte(`
schema = 1

[packs."github.com/gastownhall/gastown"]
version = "1.4.2"
commit = "abc123"
fetched = "2026-04-10T00:00:00Z"

[packs."github.com/example/tools"]
version = "2.1.0"
commit = "def456"
fetched = "2026-04-11T00:00:00Z"
`)

	entries, err := ReadPacksLock(fs, "/city")
	if err != nil {
		t.Fatalf("ReadPacksLock: %v", err)
	}

	if got := entries["github.com/gastownhall/gastown"]; got.Version != "1.4.2" || got.Commit != "abc123" || got.Fetched != "2026-04-10T00:00:00Z" {
		t.Fatalf("gastown entry = %#v, want parsed values", got)
	}
	if got := entries["github.com/example/tools"]; got.Version != "2.1.0" || got.Commit != "def456" || got.Fetched != "2026-04-11T00:00:00Z" {
		t.Fatalf("tools entry = %#v, want parsed values", got)
	}
}

func TestReadPacksLock_Track2WriterFormat(t *testing.T) {
	fs := fsys.NewFake()
	fs.Files["/city/packs.lock"] = []byte(`schema = 1

[packs."file:///tmp/repo.git//packs/base"]
version = "1.2.3"
commit = "abc123def456"
fetched = "2026-04-10T00:00:00Z"

[packs."https://github.com/example/gastown.git"]
version = "2.0.0"
commit = "fedcba654321"
fetched = "2026-04-11T00:00:00Z"
`)

	entries, err := ReadPacksLock(fs, "/city")
	if err != nil {
		t.Fatalf("ReadPacksLock: %v", err)
	}
	if got := entries[`file:///tmp/repo.git//packs/base`]; got.Commit != "abc123def456" {
		t.Fatalf("subpath entry = %#v, want parsed commit", got)
	}
	if got := entries[`https://github.com/example/gastown.git`]; got.Version != "2.0.0" {
		t.Fatalf("https entry = %#v, want parsed version", got)
	}
}

func TestReadPacksLock_UnsupportedSchemaFails(t *testing.T) {
	fs := fsys.NewFake()
	fs.Files["/city/packs.lock"] = []byte(`
schema = 2
`)

	_, err := ReadPacksLock(fs, "/city")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported schema 2") {
		t.Fatalf("error = %q, want unsupported schema", err)
	}
}

func TestCacheDir_Deterministic(t *testing.T) {
	t.Setenv("HOME", "/tmp/gc-home")

	source := "github.com/gastownhall/gastown"
	commit := "abc123def456"
	sum := sha256.Sum256([]byte(source + "\n" + commit))
	want := filepath.Join("/tmp/gc-home", ".gc", "cache", "repos", hex.EncodeToString(sum[:]))

	if got := CacheDir(source, commit); got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}

func TestRepoCacheKey_SeparatesSourceCommitBoundary(t *testing.T) {
	if RepoCacheKey("ab", "c") == RepoCacheKey("a", "bc") {
		t.Fatal("RepoCacheKey collapsed distinct source/commit pairs")
	}
}

func TestIsRemoteImportSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{source: "./packs/local", want: false},
		{source: "../packs/local", want: false},
		{source: "/packs/local", want: false},
		{source: "github.com/gastownhall/gastown", want: true},
		{source: "github.com/gastownhall/gastown//packs/base", want: true},
		{source: "gitlab.com/example/tools", want: true},
		{source: "git@github.com:org/repo.git", want: true},
		{source: "file:///tmp/repo.git//packs/base", want: true},
		{source: "https://github.com/gastownhall/gastown", want: true},
		{source: "packs/local", want: false},
	}

	for _, tt := range tests {
		if got := isRemoteImportSource(tt.source); got != tt.want {
			t.Fatalf("isRemoteImportSource(%q) = %v, want %v", tt.source, got, tt.want)
		}
	}
}

func TestParseRemoteImportSource_Subpath(t *testing.T) {
	_, subpath := parseRemoteImportSource("file:///tmp/repo.git//packs/base#v1.2.3")
	if subpath != "packs/base" {
		t.Fatalf("subpath = %q, want %q", subpath, "packs/base")
	}
}

func TestParseRemoteImportSource_GitSSHSubpath(t *testing.T) {
	_, subpath := parseRemoteImportSource("git@github.com:org/repo.git//packs/base")
	if subpath != "packs/base" {
		t.Fatalf("subpath = %q, want %q", subpath, "packs/base")
	}
}
