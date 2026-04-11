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

func marshalSiteBindings(bindings *config.SiteBindings) ([]byte, bool, error) {
	if bindings == nil || len(bindings.Rigs) == 0 {
		return nil, false, nil
	}
	data, err := bindings.Marshal()
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func writeSiteBindingsFS(fs fsys.FS, cityPath string, bindings *config.SiteBindings) error {
	path := citylayout.SiteBindingFilePath(cityPath)
	data, ok, err := marshalSiteBindings(bindings)
	if err != nil {
		return err
	}
	if !ok {
		if err := fs.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return fs.WriteFile(path, data, 0o644)
}

func writeCityAndSiteBindingsFS(fs fsys.FS, cityPath, tomlPath string, cfg *config.City, bindings *config.SiteBindings) error {
	cityData, err := cfg.Marshal()
	if err != nil {
		return err
	}
	sitePath := citylayout.SiteBindingFilePath(cityPath)
	siteData, hasSiteData, err := marshalSiteBindings(bindings)
	if err != nil {
		return err
	}

	// Commit city.toml before the live site binding so a failed city write
	// never leaves runtime behavior changed by .gc/site.toml alone.
	originalCity, err := fs.ReadFile(tomlPath)
	if err != nil {
		return err
	}
	originalSite, siteReadErr := fs.ReadFile(sitePath)
	hadSite := siteReadErr == nil
	if siteReadErr != nil && !os.IsNotExist(siteReadErr) {
		return siteReadErr
	}

	if err := fs.WriteFile(tomlPath, cityData, 0o644); err != nil {
		return err
	}

	restore := func() {
		_ = fs.WriteFile(tomlPath, originalCity, 0o644)
		switch {
		case hadSite:
			_ = fs.WriteFile(sitePath, originalSite, 0o644)
		default:
			_ = fs.Remove(sitePath)
		}
	}

	if hasSiteData {
		if err := fs.MkdirAll(filepath.Dir(sitePath), 0o755); err != nil {
			restore()
			return err
		}
		if err := fs.WriteFile(sitePath, siteData, 0o644); err != nil {
			restore()
			return err
		}
		return nil
	}
	if err := fs.Remove(sitePath); err != nil && !os.IsNotExist(err) {
		restore()
		return err
	}
	return nil
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
		existing := false
		for i := range bindings.Rigs {
			if bindings.Rigs[i].Name != rig.Name {
				continue
			}
			existing = true
			if bindings.Rigs[i].Path == "" {
				bindings.Rigs[i].Path = rig.Path
			}
			if bindings.Rigs[i].Prefix == "" {
				bindings.Rigs[i].Prefix = rig.Prefix
			}
			if bindings.Rigs[i].Suspended == nil && rig.Suspended {
				bindings.Rigs[i].Suspended = rigBindingBoolPtr(true)
			}
			break
		}
		if existing {
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
