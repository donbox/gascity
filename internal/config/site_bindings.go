package config

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/gastownhall/gascity/internal/citylayout"
	"github.com/gastownhall/gascity/internal/fsys"
)

// SiteBindings captures machine-local config that lives under .gc/ rather than
// in the checked-in city.toml surface.
type SiteBindings struct {
	Rigs []RigSiteBinding `toml:"rigs,omitempty"`
}

// RigSiteBinding captures machine-local rig binding state.
type RigSiteBinding struct {
	Name      string `toml:"name"`
	Path      string `toml:"path,omitempty"`
	Prefix    string `toml:"prefix,omitempty"`
	Suspended *bool  `toml:"suspended,omitempty"`
}

// LoadSiteBindings reads .gc/site.toml if present. Missing files return an
// empty binding set.
func LoadSiteBindings(fs fsys.FS, cityRoot string) (*SiteBindings, error) {
	path := citylayout.SiteBindingFilePath(cityRoot)
	data, err := fs.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SiteBindings{}, nil
		}
		return nil, fmt.Errorf("loading site bindings %q: %w", path, err)
	}

	var bindings SiteBindings
	if _, err := toml.Decode(string(data), &bindings); err != nil {
		return nil, fmt.Errorf("parsing site bindings %q: %w", path, err)
	}
	return &bindings, nil
}

// Marshal encodes site bindings to TOML bytes in stable name order.
func (s *SiteBindings) Marshal() ([]byte, error) {
	clone := *s
	if len(clone.Rigs) > 0 {
		clone.Rigs = append([]RigSiteBinding(nil), clone.Rigs...)
		sort.Slice(clone.Rigs, func(i, j int) bool {
			return clone.Rigs[i].Name < clone.Rigs[j].Name
		})
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	if err := enc.Encode(clone); err != nil {
		return nil, fmt.Errorf("marshaling site bindings: %w", err)
	}
	return buf.Bytes(), nil
}

// ApplySiteBindings overlays machine-local rig bindings onto the composed city
// config. Bindings only affect rigs already declared in city.toml/pack config.
func ApplySiteBindings(cfg *City, bindings *SiteBindings) {
	if cfg == nil || bindings == nil || len(bindings.Rigs) == 0 {
		return
	}

	byName := make(map[string]RigSiteBinding, len(bindings.Rigs))
	for _, binding := range bindings.Rigs {
		if binding.Name == "" {
			continue
		}
		byName[binding.Name] = binding
	}

	for i := range cfg.Rigs {
		binding, ok := byName[cfg.Rigs[i].Name]
		if !ok {
			continue
		}
		if binding.Path != "" {
			cfg.Rigs[i].Path = binding.Path
		}
		if binding.Prefix != "" {
			cfg.Rigs[i].Prefix = binding.Prefix
		}
		if binding.Suspended != nil {
			cfg.Rigs[i].Suspended = *binding.Suspended
		}
	}
}
