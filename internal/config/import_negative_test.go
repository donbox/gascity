package config

// Negative and stress tests for the V2 import system.
// These test error paths, malformed inputs, and boundary conditions.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/fsys"
)

func TestImport_MalformedAgentToml(t *testing.T) {
	// A malformed agent.toml in agents/<name>/ should produce a clear error.
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")
	agentDir := filepath.Join(packDir, "agents", "bad")
	os.MkdirAll(agentDir, 0o755)

	writeTestFile(t, packDir, "pack.toml", `
[pack]
name = "mypk"
schema = 1
`)
	writeTestFile(t, agentDir, "agent.toml", `{{invalid toml`)

	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)
	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"
includes = ["../mypk"]
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err == nil {
		t.Fatal("expected error for malformed agent.toml")
	}
	if !strings.Contains(err.Error(), "agents/bad/agent.toml") {
		t.Errorf("error should mention agent.toml path; got: %v", err)
	}
}

func TestImport_InvalidPackSchemaInCityPackToml(t *testing.T) {
	// A city pack.toml with invalid schema should produce a clear error.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)

	writeTestFile(t, cityDir, "pack.toml", `
[pack]
name = "test"
schema = 999
`)
	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err == nil {
		t.Fatal("expected error for invalid pack schema")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("error should mention schema; got: %v", err)
	}
}

func TestImport_MalformedCityPackToml(t *testing.T) {
	// A malformed city pack.toml should produce a clear error, not be
	// silently ignored.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)

	writeTestFile(t, cityDir, "pack.toml", `{{not valid toml`)
	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err == nil {
		t.Fatal("expected error for malformed city pack.toml")
	}
	if !strings.Contains(err.Error(), "pack.toml") {
		t.Errorf("error should mention pack.toml; got: %v", err)
	}
}

func TestImport_TransitiveFalseWithExport(t *testing.T) {
	// transitive=false on an import should suppress its nested deps
	// even if the nested pack uses export=true internally.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	outerDir := filepath.Join(dir, "outer")
	innerDir := filepath.Join(dir, "inner")
	deepDir := filepath.Join(dir, "deep")

	for _, d := range []string{cityDir, outerDir, innerDir, deepDir} {
		os.MkdirAll(d, 0o755)
	}

	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"

[imports.outer]
source = "../outer"
transitive = false
`)
	writeTestFile(t, outerDir, "pack.toml", `
[pack]
name = "outer"
schema = 1

[imports.inner]
source = "../inner"
export = true

[[agent]]
name = "outer-agent"
scope = "city"
`)
	writeTestFile(t, innerDir, "pack.toml", `
[pack]
name = "inner"
schema = 1

[imports.deep]
source = "../deep"

[[agent]]
name = "inner-agent"
scope = "city"
`)
	writeTestFile(t, deepDir, "pack.toml", `
[pack]
name = "deep"
schema = 1

[[agent]]
name = "deep-agent"
scope = "city"
`)

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	explicit := explicitAgents(cfg.Agents)
	found := map[string]bool{}
	for _, a := range explicit {
		found[a.QualifiedName()] = true
	}

	// outer-agent should be present (direct from outer).
	if !found["outer.outer-agent"] {
		t.Errorf("missing outer.outer-agent; got: %v", found)
	}
	// inner-agent and deep-agent should NOT be present (transitive=false
	// on the city's import of outer suppresses all nested deps).
	for qn := range found {
		if strings.Contains(qn, "inner-agent") || strings.Contains(qn, "deep-agent") {
			t.Errorf("transitive=false should block nested agents; got: %v", found)
			break
		}
	}
}

func TestImport_DeeplyNestedChain(t *testing.T) {
	// A→B→C→D→E: five-level import chain should work.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	packs := []string{"a", "b", "c", "d", "e"}

	os.MkdirAll(cityDir, 0o755)
	for _, name := range packs {
		os.MkdirAll(filepath.Join(dir, name), 0o755)
	}

	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"

[imports.a]
source = "../a"
`)

	for i, name := range packs {
		var importLine string
		if i < len(packs)-1 {
			next := packs[i+1]
			importLine = "[imports." + next + "]\nsource = \"../" + next + "\"\n"
		}
		writeTestFile(t, filepath.Join(dir, name), "pack.toml", `
[pack]
name = "`+name+`"
schema = 1

`+importLine+`

[[agent]]
name = "`+name+`-agent"
scope = "city"
`)
	}

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	explicit := explicitAgents(cfg.Agents)
	found := map[string]bool{}
	for _, a := range explicit {
		found[a.Name] = true
	}

	for _, name := range packs {
		agentName := name + "-agent"
		if !found[agentName] {
			t.Errorf("missing %s from deep chain; got: %v", agentName, found)
		}
	}
}

func TestImport_ManyImports(t *testing.T) {
	// A city with 20 imports should work without issues.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)

	var importLines []string
	for i := 0; i < 20; i++ {
		name := "pack" + string(rune('a'+i))
		packDir := filepath.Join(dir, name)
		os.MkdirAll(packDir, 0o755)
		writeTestFile(t, packDir, "pack.toml", `
[pack]
name = "`+name+`"
schema = 1

[[agent]]
name = "agent"
scope = "city"
`)
		importLines = append(importLines, "[imports."+name+"]\nsource = \"../"+name+"\"")
	}

	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"

`+strings.Join(importLines, "\n\n"))

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	explicit := explicitAgents(cfg.Agents)
	if len(explicit) != 20 {
		t.Errorf("expected 20 agents from 20 imports, got %d", len(explicit))
	}
}

func TestImport_EmptyImportSource(t *testing.T) {
	// An import with empty source should produce a clear error.
	dir := t.TempDir()
	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)

	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"

[imports.bad]
source = ""
`)

	_, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err == nil {
		t.Fatal("expected error for empty import source")
	}
}

func TestImport_AgentDiscoveryWithNoPromptOrToml(t *testing.T) {
	// An agents/<name>/ directory with neither prompt.md nor agent.toml
	// should still create an agent (minimal discovery).
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")
	agentDir := filepath.Join(packDir, "agents", "minimal")
	os.MkdirAll(agentDir, 0o755)

	writeTestFile(t, packDir, "pack.toml", `
[pack]
name = "mypk"
schema = 1
`)
	// Create a file inside the agent dir (not prompt.md or agent.toml)
	// to prove the dir exists but has no standard files.
	writeTestFile(t, agentDir, "notes.txt", "just a note")

	cityDir := filepath.Join(dir, "city")
	os.MkdirAll(cityDir, 0o755)
	writeTestFile(t, cityDir, "city.toml", `
[workspace]
name = "test"
includes = ["../mypk"]
`)

	cfg, _, err := LoadWithIncludes(fsys.OSFS{}, filepath.Join(cityDir, "city.toml"))
	if err != nil {
		t.Fatalf("LoadWithIncludes: %v", err)
	}

	explicit := explicitAgents(cfg.Agents)
	found := false
	for _, a := range explicit {
		if a.Name == "minimal" {
			found = true
			if a.PromptTemplate != "" {
				t.Errorf("minimal agent should have no prompt template, got %q", a.PromptTemplate)
			}
			break
		}
	}
	if !found {
		t.Error("minimal agent should still be discovered from empty agents/ subdir")
	}
}
