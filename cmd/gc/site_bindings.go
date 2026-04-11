package main

import (
	"os"
	"path/filepath"

	"github.com/gastownhall/gascity/internal/citylayout"
	"github.com/gastownhall/gascity/internal/config"
	"github.com/gastownhall/gascity/internal/fsys"
)

func loadSiteBindingsFS(fs fsys.FS, cityPath string) (*config.SiteBindings, error) {
	return config.LoadSiteBindings(fs, cityPath)
}

func writeSiteBindingsFS(fs fsys.FS, cityPath string, bindings *config.SiteBindings) error {
	path := citylayout.SiteBindingFilePath(cityPath)
	if bindings == nil || len(bindings.Rigs) == 0 {
		if err := fs.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := bindings.Marshal()
	if err != nil {
		return err
	}
	return fs.WriteFile(path, data, 0o644)
}

func seedSiteBindingsFromConfig(bindings *config.SiteBindings, cfg *config.City) {
	if bindings == nil || cfg == nil {
		return
	}
	for _, rig := range cfg.Rigs {
		if rig.Name == "" {
			continue
		}
		if rig.Path == "" && rig.Prefix == "" && !rig.Suspended {
			continue
		}
		upsertRigSiteBinding(bindings, config.RigSiteBinding{
			Name:   rig.Name,
			Path:   rig.Path,
			Prefix: rig.Prefix,
		})
		if rig.Suspended {
			setRigBindingSuspended(bindings, rig.Name, true)
		}
	}
}

func upsertRigSiteBinding(bindings *config.SiteBindings, binding config.RigSiteBinding) {
	if bindings == nil || binding.Name == "" {
		return
	}
	for i := range bindings.Rigs {
		if bindings.Rigs[i].Name != binding.Name {
			continue
		}
		if binding.Path != "" {
			bindings.Rigs[i].Path = binding.Path
		}
		if binding.Prefix != "" {
			bindings.Rigs[i].Prefix = binding.Prefix
		}
		return
	}
	bindings.Rigs = append(bindings.Rigs, binding)
}

func setRigBindingSuspended(bindings *config.SiteBindings, rigName string, suspended bool) {
	if bindings == nil || rigName == "" {
		return
	}
	for i := range bindings.Rigs {
		if bindings.Rigs[i].Name != rigName {
			continue
		}
		bindings.Rigs[i].Suspended = rigBindingBoolPtr(suspended)
		return
	}
	bindings.Rigs = append(bindings.Rigs, config.RigSiteBinding{
		Name:      rigName,
		Suspended: rigBindingBoolPtr(suspended),
	})
}

func removeRigSiteBinding(bindings *config.SiteBindings, rigName string) {
	if bindings == nil || rigName == "" {
		return
	}
	filtered := bindings.Rigs[:0]
	for _, binding := range bindings.Rigs {
		if binding.Name == rigName {
			continue
		}
		filtered = append(filtered, binding)
	}
	bindings.Rigs = filtered
}

func canonicalizeRigBindings(cfg *config.City) {
	if cfg == nil {
		return
	}
	for i := range cfg.Rigs {
		cfg.Rigs[i].Path = ""
		cfg.Rigs[i].Prefix = ""
		cfg.Rigs[i].Suspended = false
	}
}

func cityWithSiteBindings(cfg *config.City, bindings *config.SiteBindings) *config.City {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	if len(cfg.Rigs) > 0 {
		clone.Rigs = append([]config.Rig(nil), cfg.Rigs...)
	}
	config.ApplySiteBindings(&clone, bindings)
	return &clone
}

func findRigByName(rigs []config.Rig, name string) *config.Rig {
	for i := range rigs {
		if rigs[i].Name == name {
			return &rigs[i]
		}
	}
	return nil
}

func rigBindingBoolPtr(v bool) *bool { return &v }
