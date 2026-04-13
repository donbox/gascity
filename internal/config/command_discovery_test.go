package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/fsys"
)

func TestDiscoverPackCommands_BasicAndNested(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/status/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/status/help.md", "status help")
	writeTestFile(t, packDir, "commands/repo/sync/run.sh", "#!/bin/sh\nexit 0\n")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d commands, want 2", len(got))
	}

	if got[0].Name != "repo/sync" {
		t.Fatalf("got first command %q, want %q", got[0].Name, "repo/sync")
	}
	if !reflect.DeepEqual(got[0].Command, []string{"repo", "sync"}) {
		t.Fatalf("repo/sync words = %#v, want %#v", got[0].Command, []string{"repo", "sync"})
	}

	if got[1].Name != "status" {
		t.Fatalf("got second command %q, want %q", got[1].Name, "status")
	}
	if !reflect.DeepEqual(got[1].Command, []string{"status"}) {
		t.Fatalf("status words = %#v, want %#v", got[1].Command, []string{"status"})
	}
	if got[1].HelpFile == "" {
		t.Fatal("status HelpFile = empty, want discovered help.md")
	}
}

func TestDiscoverPackCommands_ManifestOverride(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/repo-sync/command.toml", `
description = "Sync the repo"
run = "entry.sh"
`)
	writeTestFile(t, packDir, "commands/repo-sync/entry.sh", "#!/bin/sh\nexit 0\n")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d commands, want 1", len(got))
	}
	if !reflect.DeepEqual(got[0].Command, []string{"repo-sync"}) {
		t.Fatalf("command words = %#v, want %#v", got[0].Command, []string{"repo-sync"})
	}
	if got[0].Description != "Sync the repo" {
		t.Fatalf("description = %q, want %q", got[0].Description, "Sync the repo")
	}
	wantRun := filepath.Join(packDir, "commands", "repo-sync", "entry.sh")
	if got[0].RunScript != wantRun {
		t.Fatalf("RunScript = %q, want %q", got[0].RunScript, wantRun)
	}
}

func TestDiscoverPackCommands_SkipsHiddenAndUnderscoreDirs(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/status/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/.hidden/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/_private/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/repo/_internal/run.sh", "#!/bin/sh\nexit 0\n")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d commands, want 1", len(got))
	}
	if got[0].Name != "status" {
		t.Fatalf("got command %q, want %q", got[0].Name, "status")
	}
}

func TestDiscoverPackCommands_NoCommandsDir(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d commands, want 0", len(got))
	}
}

func TestDiscoverPackCommands_BadManifest(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/status/command.toml", "command = [")
	writeTestFile(t, packDir, "commands/status/run.sh", "#!/bin/sh\nexit 0\n")

	_, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err == nil {
		t.Fatal("DiscoverPackCommands error = nil, want manifest parse error")
	}
}

func TestDiscoverPackCommands_AllowsRunnableParentAndChild(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/repo/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/repo/sync/run.sh", "#!/bin/sh\nexit 0\n")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d commands, want 2", len(got))
	}
	if !reflect.DeepEqual(got[0].Command, []string{"repo"}) {
		t.Fatalf("first command words = %#v, want %#v", got[0].Command, []string{"repo"})
	}
	if !reflect.DeepEqual(got[1].Command, []string{"repo", "sync"}) {
		t.Fatalf("second command words = %#v, want %#v", got[1].Command, []string{"repo", "sync"})
	}
}

func TestDiscoverPackCommands_AllowsVisibleAssetSubdirsUnderLeaf(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/deploy/run.sh", "#!/bin/sh\nexit 0\n")
	writeTestFile(t, packDir, "commands/deploy/templates/example.txt", "template")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d commands, want 1", len(got))
	}
	if !reflect.DeepEqual(got[0].Command, []string{"deploy"}) {
		t.Fatalf("command words = %#v, want %#v", got[0].Command, []string{"deploy"})
	}
}

func TestDiscoverPackCommands_HelpOnlyParentNode(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/repo/help.md", "repo help")
	writeTestFile(t, packDir, "commands/repo/sync/run.sh", "#!/bin/sh\nexit 0\n")

	got, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err != nil {
		t.Fatalf("DiscoverPackCommands: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d commands, want 2", len(got))
	}
	if !reflect.DeepEqual(got[0].Command, []string{"repo"}) {
		t.Fatalf("first command words = %#v, want %#v", got[0].Command, []string{"repo"})
	}
	if got[0].RunScript != "" {
		t.Fatalf("repo RunScript = %q, want empty for help-only node", got[0].RunScript)
	}
	if got[0].HelpFile == "" {
		t.Fatal("repo HelpFile = empty, want discovered help.md")
	}
	if !reflect.DeepEqual(got[1].Command, []string{"repo", "sync"}) {
		t.Fatalf("second command words = %#v, want %#v", got[1].Command, []string{"repo", "sync"})
	}
}

func TestDiscoverPackCommands_RejectsRemovedCommandField(t *testing.T) {
	dir := t.TempDir()
	packDir := filepath.Join(dir, "mypk")

	writeTestFile(t, packDir, "commands/repo-sync/command.toml", `
command = ["repo", "sync"]
run = "run.sh"
`)
	writeTestFile(t, packDir, "commands/repo-sync/run.sh", "#!/bin/sh\nexit 0\n")

	_, err := DiscoverPackCommands(fsys.OSFS{}, packDir, "mypk")
	if err == nil {
		t.Fatal("DiscoverPackCommands error = nil, want unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error = %q, want unknown field message", err.Error())
	}
	if !strings.Contains(err.Error(), "command") {
		t.Fatalf("error = %q, want mention of removed command field", err.Error())
	}
}
