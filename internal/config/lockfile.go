package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gastownhall/gascity/internal/fsys"
)

// LockEntry records a resolved remote import in packs.lock.
type LockEntry struct {
	Version string `toml:"version"`
	Commit  string `toml:"commit"`
	Fetched string `toml:"fetched"`
}

type packsLockFile struct {
	Schema int                  `toml:"schema"`
	Packs  map[string]LockEntry `toml:"packs"`
}

const packsLockSchema = 1

// ReadPacksLock reads packs.lock from the city root.
// Missing lock files are treated as empty lock state.
func ReadPacksLock(fs fsys.FS, cityRoot string) (map[string]LockEntry, error) {
	path := filepath.Join(cityRoot, "packs.lock")
	data, err := fs.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]LockEntry{}, nil
		}
		return nil, fmt.Errorf("reading packs.lock: %w", err)
	}

	var lock packsLockFile
	if _, err := toml.Decode(string(data), &lock); err != nil {
		return nil, fmt.Errorf("parsing packs.lock: %w", err)
	}
	if lock.Schema != 0 && lock.Schema != packsLockSchema {
		return nil, fmt.Errorf("parsing packs.lock: unsupported schema %d", lock.Schema)
	}
	if lock.Packs == nil {
		return map[string]LockEntry{}, nil
	}
	return lock.Packs, nil
}

// CacheDir returns the shared repo cache directory for a source+commit pair.
func CacheDir(source, commit string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
		if home == "" {
			home = os.TempDir()
		}
	}
	sum := sha256.Sum256([]byte(source + "\n" + commit))
	return filepath.Join(home, ".gc", "cache", "repos", hex.EncodeToString(sum[:]))
}

func resolveImportRef(fs fsys.FS, source, declDir, cityRoot string, locks map[string]LockEntry) (string, error) {
	if !isRemoteImportSource(source) {
		return resolveConfigPath(source, declDir, cityRoot), nil
	}

	entry, ok := locks[source]
	if !ok {
		return "", fmt.Errorf("source %q: not in packs.lock - run gc import install", source)
	}

	cacheDir := CacheDir(source, entry.Commit)
	if _, err := fs.Stat(cacheDir); err != nil {
		return "", fmt.Errorf("source %q: cache missing - run gc import install", source)
	}

	return cacheDir, nil
}

func isRemoteImportSource(source string) bool {
	if source == "" {
		return false
	}
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") || strings.HasPrefix(source, "/") {
		return false
	}
	if strings.HasPrefix(source, "git@") ||
		strings.HasPrefix(source, "ssh://") ||
		strings.HasPrefix(source, "https://") ||
		strings.HasPrefix(source, "http://") ||
		strings.HasPrefix(source, "file://") {
		return true
	}
	if strings.Contains(source, "github.com/") || strings.Contains(source, "gitlab.com/") {
		return true
	}
	slash := strings.IndexByte(source, '/')
	if slash <= 0 {
		return false
	}
	return strings.Contains(source[:slash], ".")
}
