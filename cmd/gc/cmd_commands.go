package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gastownhall/gascity/internal/citylayout"
	"github.com/gastownhall/gascity/internal/config"
	"github.com/spf13/cobra"
)

func addDiscoveredCommandsToRoot(root *cobra.Command, entries []config.DiscoveredCommand, cityPath, cityName string, stdout, stderr io.Writer) {
	core := coreCommandNames(root)
	grouped := make(map[string][]config.DiscoveredCommand)
	for _, entry := range entries {
		if entry.BindingName == "" {
			continue
		}
		grouped[entry.BindingName] = append(grouped[entry.BindingName], entry)
	}

	bindings := make([]string, 0, len(grouped))
	for binding := range grouped {
		bindings = append(bindings, binding)
	}
	slices.Sort(bindings)

	for _, binding := range bindings {
		if core[binding] {
			fmt.Fprintf(stderr, "gc: import binding %q: name shadows core command, skipping\n", binding) //nolint:errcheck
			continue
		}
		nsCmd := newDiscoveredNamespaceCmd(binding, grouped[binding], cityPath, cityName, stdout, stderr)
		root.AddCommand(nsCmd)
	}
}

func newDiscoveredNamespaceCmd(binding string, entries []config.DiscoveredCommand, cityPath, cityName string, stdout, stderr io.Writer) *cobra.Command {
	ns := newDiscoveredTreeCommand(binding)
	ns.Short = fmt.Sprintf("Commands from the %s import", binding)

	for _, entry := range sortCommandsForTree(entries) {
		addOrMergeDiscoveredCommand(ns, entry, cityPath, cityName, stdout, stderr)
	}

	return ns
}

func addOrMergeDiscoveredCommand(root *cobra.Command, entry config.DiscoveredCommand, cityPath, cityName string, stdout, stderr io.Writer) {
	if len(entry.Command) == 0 {
		return
	}

	node := root
	for _, word := range entry.Command {
		node = ensureDiscoveredTreeNode(node, word)
	}
	applyDiscoveredCommandNode(node, entry, cityPath, cityName, stdout, stderr)
}

func ensureDiscoveredTreeNode(parent *cobra.Command, word string) *cobra.Command {
	if existing := findSubcommand(parent, word); existing != nil {
		return existing
	}
	next := newDiscoveredTreeCommand(word)
	parent.AddCommand(next)
	return next
}

func newDiscoveredTreeCommand(word string) *cobra.Command {
	return &cobra.Command{
		Use: word,
		RunE: func(c *cobra.Command, _ []string) error {
			return c.Help()
		},
	}
}

func applyDiscoveredCommandNode(node *cobra.Command, entry config.DiscoveredCommand, cityPath, cityName string, stdout, stderr io.Writer) {
	if entry.Description != "" {
		node.Short = entry.Description
	}
	if long := readDiscoveredHelp(entry.HelpFile); long != "" {
		node.Long = long
	}
	if entry.RunScript == "" {
		return
	}

	entryCopy := entry
	node.DisableFlagParsing = true
	node.RunE = func(_ *cobra.Command, args []string) error {
		code := runDiscoveredCommand(entryCopy, cityPath, cityName, args, stdin(), stdout, stderr)
		if code != 0 {
			os.Exit(code)
		}
		return nil
	}
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, existing := range cmd.Commands() {
		if existing.Name() == name {
			return existing
		}
	}
	return nil
}

func readDiscoveredHelp(helpFile string) string {
	if helpFile == "" {
		return ""
	}
	data, err := os.ReadFile(helpFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func runDiscoveredCommand(entry config.DiscoveredCommand, cityPath, cityName string, args []string, stdinR io.Reader, stdout, stderr io.Writer) int {
	packDir := entry.PackDir
	if packDir == "" {
		packDir = packRootFromEntryDir(entry.SourceDir, "commands")
	}
	scriptPath := expandScriptTemplate(entry.RunScript, cityPath, cityName, packDir)
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(entry.SourceDir, scriptPath)
	}

	cmd := exec.Command(scriptPath, args...)
	cmd.Stdin = stdinR
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(), citylayout.PackRuntimeEnv(cityPath, entry.PackName)...)
	cmd.Env = append(cmd.Env,
		"GC_PACK_DIR="+packDir,
		"GC_PACK_NAME="+entry.PackName,
		"GC_CITY_NAME="+cityName,
	)

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "gc %s %s: %v\n", entry.BindingName, strings.Join(entry.Command, " "), err) //nolint:errcheck
		return 1
	}
	return 0
}

func tryDiscoveredCommandFallback(args []string, cfg *config.City, cityPath string, stdout, stderr io.Writer) bool {
	if len(args) == 0 {
		return false
	}

	binding := args[0]
	var matching []config.DiscoveredCommand
	for _, entry := range cfg.PackCommands {
		if entry.BindingName == binding {
			matching = append(matching, entry)
		}
	}
	if len(matching) == 0 {
		return false
	}

	if len(args) == 1 {
		fmt.Fprintf(stdout, "Available commands for %s:\n", binding) //nolint:errcheck
		for _, entry := range matching {
			fmt.Fprintf(stdout, "  %-20s %s\n", strings.Join(entry.Command, " "), entry.Description) //nolint:errcheck
		}
		return true
	}

	commandWords := args[1:]
	if exact, ok := exactDiscoveredCommandMatch(matching, commandWords); ok {
		if exact.RunScript != "" {
			code := runDiscoveredCommand(exact, cityPath, cfg.Workspace.Name, nil, stdin(), stdout, stderr)
			if code != 0 {
				os.Exit(code)
			}
			return true
		}
		printDiscoveredCommandNodeHelp(binding, exact, matching, stdout)
		return true
	}

	cityName := cfg.Workspace.Name
	sort.SliceStable(matching, func(i, j int) bool {
		return len(matching[i].Command) > len(matching[j].Command)
	})
	for _, entry := range matching {
		if entry.RunScript == "" {
			continue
		}
		if len(args)-1 < len(entry.Command) {
			continue
		}
		if slices.Equal(args[1:1+len(entry.Command)], entry.Command) {
			code := runDiscoveredCommand(entry, cityPath, cityName, args[1+len(entry.Command):], stdin(), stdout, stderr)
			if code != 0 {
				os.Exit(code)
			}
			return true
		}
	}

	return false
}

func exactDiscoveredCommandMatch(entries []config.DiscoveredCommand, command []string) (config.DiscoveredCommand, bool) {
	for _, entry := range entries {
		if slices.Equal(entry.Command, command) {
			return entry, true
		}
	}
	return config.DiscoveredCommand{}, false
}

func printDiscoveredCommandNodeHelp(binding string, entry config.DiscoveredCommand, entries []config.DiscoveredCommand, stdout io.Writer) {
	if long := readDiscoveredHelp(entry.HelpFile); long != "" {
		fmt.Fprintln(stdout, long) //nolint:errcheck
	}

	children := discoveredImmediateChildren(entries, entry.Command)
	if len(children) == 0 {
		return
	}

	prefix := strings.TrimSpace(strings.Join(append([]string{binding}, entry.Command...), " "))
	fmt.Fprintf(stdout, "Available subcommands for %s:\n", prefix) //nolint:errcheck
	for _, child := range children {
		fmt.Fprintf(stdout, "  %s\n", child) //nolint:errcheck
	}
}

func discoveredImmediateChildren(entries []config.DiscoveredCommand, prefix []string) []string {
	seen := make(map[string]bool)
	var children []string
	for _, entry := range entries {
		if len(entry.Command) != len(prefix)+1 {
			continue
		}
		if !slices.Equal(entry.Command[:len(prefix)], prefix) {
			continue
		}
		child := entry.Command[len(prefix)]
		if seen[child] {
			continue
		}
		seen[child] = true
		children = append(children, child)
	}
	sort.Strings(children)
	return children
}

func sortCommandsForTree(entries []config.DiscoveredCommand) []config.DiscoveredCommand {
	sorted := append([]config.DiscoveredCommand(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if len(sorted[i].Command) != len(sorted[j].Command) {
			return len(sorted[i].Command) < len(sorted[j].Command)
		}
		return strings.Join(sorted[i].Command, "\x00") < strings.Join(sorted[j].Command, "\x00")
	})
	return sorted
}

func packRootFromEntryDir(sourceDir, topLevel string) string {
	marker := string(filepath.Separator) + topLevel + string(filepath.Separator)
	if idx := strings.LastIndex(sourceDir, marker); idx >= 0 {
		return sourceDir[:idx]
	}
	return filepath.Dir(sourceDir)
}
