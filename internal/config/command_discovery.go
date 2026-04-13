package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gastownhall/gascity/internal/fsys"
)

// DiscoveredCommand is a convention-discovered pack command.
type DiscoveredCommand struct {
	Name        string
	Command     []string
	Description string
	RunScript   string
	HelpFile    string
	SourceDir   string
	PackDir     string
	PackName    string
	BindingName string
}

type commandManifest struct {
	Description string `toml:"description"`
	Run         string `toml:"run"`
}

// DiscoverPackCommands scans a pack's commands/ tree and returns
// convention-discovered command nodes. Nested directories imply nested
// command words by default. A node may be runnable, a parent, or both.
func DiscoverPackCommands(fs fsys.FS, packDir, packName string) ([]DiscoveredCommand, error) {
	commandsDir := filepath.Join(packDir, "commands")
	if _, err := fs.Stat(commandsDir); err != nil {
		return nil, nil
	}

	var discovered []DiscoveredCommand
	if err := walkCommandDirs(fs, packDir, packName, commandsDir, nil, &discovered); err != nil {
		return nil, err
	}
	return discovered, nil
}

func walkCommandDirs(fs fsys.FS, packDir, packName, dir string, words []string, discovered *[]DiscoveredCommand) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}

		childDir := filepath.Join(dir, name)
		childWords := append(append([]string{}, words...), name)

		cmd, ok, err := discoveredCommandFromDir(fs, packDir, packName, childDir, childWords)
		if err != nil {
			return err
		}
		if ok {
			*discovered = append(*discovered, cmd)
		}

		if err := walkCommandDirs(fs, packDir, packName, childDir, childWords, discovered); err != nil {
			return err
		}
	}

	return nil
}

func discoveredCommandFromDir(fs fsys.FS, packDir, packName, commandDir string, defaultWords []string) (DiscoveredCommand, bool, error) {
	runRel := "run.sh"
	helpPath := filepath.Join(commandDir, "help.md")
	manifestPath := filepath.Join(commandDir, "command.toml")
	words := append([]string{}, defaultWords...)
	description := ""

	if data, err := fs.ReadFile(manifestPath); err == nil {
		var manifest commandManifest
		md, err := toml.Decode(string(data), &manifest)
		if err != nil {
			rel, _ := filepath.Rel(filepath.Join(packDir, "commands"), manifestPath)
			return DiscoveredCommand{}, false, fmt.Errorf("commands/%s: %w", filepath.ToSlash(rel), err)
		}
		rel, _ := filepath.Rel(filepath.Join(packDir, "commands"), manifestPath)
		source := filepath.ToSlash(filepath.Join("commands", rel))
		if err := manifestUndecodedError(md, source); err != nil {
			return DiscoveredCommand{}, false, err
		}
		if manifest.Description != "" {
			description = manifest.Description
		}
		if manifest.Run != "" {
			runRel = manifest.Run
		}
	}

	runPath := filepath.Join(commandDir, runRel)
	runnable := false
	if strings.Contains(runRel, "{{") {
		runPath = runRel
		runnable = true
	}
	if !runnable {
		if _, err := fs.Stat(runPath); err == nil {
			runnable = true
		} else {
			runPath = ""
		}
	}

	helpFile := ""
	if _, err := fs.Stat(helpPath); err == nil {
		helpFile = helpPath
	}
	if !runnable && helpFile == "" && description == "" {
		return DiscoveredCommand{}, false, nil
	}

	relDir, err := filepath.Rel(filepath.Join(packDir, "commands"), commandDir)
	if err != nil {
		relDir = commandDir
	}

	return DiscoveredCommand{
		Name:        filepath.ToSlash(relDir),
		Command:     append([]string{}, words...),
		Description: description,
		RunScript:   runPath,
		HelpFile:    helpFile,
		SourceDir:   commandDir,
		PackDir:     packDir,
		PackName:    packName,
	}, true, nil
}

func manifestUndecodedError(md toml.MetaData, source string) error {
	undecoded := md.Undecoded()
	if len(undecoded) == 0 {
		return nil
	}
	names := make([]string, 0, len(undecoded))
	for _, key := range undecoded {
		names = append(names, key.String())
	}
	return fmt.Errorf("%s: unknown field(s): %s", source, strings.Join(names, ", "))
}
