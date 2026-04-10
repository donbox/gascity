package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/fsys"
)

func TestLoadWithIncludes_RemoteImportUsesPacksLockTransitively(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cityRoot := t.TempDir()
	writeFile(t, cityRoot, "city.toml", `
[workspace]
name = "test"

[imports.gastown]
source = "github.com/gastownhall/gastown"
`)
	writeFile(t, cityRoot, "packs.lock", `
schema = 1

[packs."github.com/gastownhall/gastown"]
version = "1.4.2"
commit = "abc123"
fetched = "2026-04-10T00:00:00Z"

[packs."github.com/example/tools"]
version = "2.1.0"
commit = "def456"
fetched = "2026-04-10T00:00:00Z"
`)

	gastownDir := CacheDir("github.com/gastownhall/gastown", "abc123")
	if err := os.MkdirAll(gastownDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, gastownDir, "pack.toml", `
[pack]
name = "gastown"
schema = 1

[imports.tools]
source = "github.com/example/tools"

[[agent]]
name = "mayor"
`)

	toolsDir := CacheDir("github.com/example/tools", "def456")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, toolsDir, "pack.toml", `
[pack]
name = "tools"
schema = 1

[[agent]]
name = "wrench"
`)

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityRoot, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	var names []string
	for _, agent := range cfg.Agents {
		names = append(names, agent.QualifiedName())
	}
	if !containsString(names, "gastown.mayor") {
		t.Fatalf("qualified agents = %v, want gastown.mayor", names)
	}
	if !containsString(names, "gastown.wrench") {
		t.Fatalf("qualified agents = %v, want transitive gastown.wrench", names)
	}
}

func TestLoadWithIncludes_V1IncludeCanUsePacksLockForNestedImport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cityRoot := t.TempDir()
	writeFile(t, cityRoot, "city.toml", `
include = ["fragments/packs.toml"]

[workspace]
name = "test"
`)
	writeFile(t, cityRoot, "fragments/packs.toml", `
[workspace]
includes = ["fragments/packs/base"]
`)
	writeFile(t, cityRoot, "fragments/packs/base/pack.toml", `
[pack]
name = "base"
schema = 1

[imports.tools]
source = "github.com/example/tools"

[[agent]]
name = "foreman"
`)
	writeFile(t, cityRoot, "packs.lock", `
schema = 1

[packs."github.com/example/tools"]
version = "2.1.0"
commit = "def456"
fetched = "2026-04-10T00:00:00Z"
`)

	toolsDir := CacheDir("github.com/example/tools", "def456")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, toolsDir, "pack.toml", `
[pack]
name = "tools"
schema = 1

[[agent]]
name = "wrench"
`)

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityRoot, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	var names []string
	for _, agent := range cfg.Agents {
		names = append(names, agent.QualifiedName())
	}
	if !containsString(names, "tools.wrench") {
		t.Fatalf("qualified agents = %v, want nested import agent", names)
	}
}

func TestLoadWithIncludes_RemoteImportMissingLockFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cityRoot := t.TempDir()
	writeFile(t, cityRoot, "city.toml", `
[workspace]
name = "test"

[imports.gastown]
source = "github.com/gastownhall/gastown"
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityRoot, "city.toml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `source "github.com/gastownhall/gastown": not in packs.lock`) {
		t.Fatalf("error = %q, want missing lock guidance", err)
	}
}

func TestLoadWithIncludes_RemoteImportMissingCacheFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cityRoot := t.TempDir()
	writeFile(t, cityRoot, "city.toml", `
[workspace]
name = "test"

[imports.gastown]
source = "github.com/gastownhall/gastown"
`)
	writeFile(t, cityRoot, "packs.lock", `
schema = 1

[packs."github.com/gastownhall/gastown"]
version = "1.4.2"
commit = "abc123"
fetched = "2026-04-10T00:00:00Z"
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityRoot, "city.toml"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `source "github.com/gastownhall/gastown": cache missing`) {
		t.Fatalf("error = %q, want missing cache guidance", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
