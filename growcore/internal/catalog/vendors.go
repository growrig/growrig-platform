package catalog

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// Vendor is a manufacturer represented in the device catalogue. Logo is an
// optional path relative to the vendor directory. Color and Background style
// the fallback shown until real artwork is supplied.
type Vendor struct {
	ID         string `json:"id"`
	Name       string `json:"name" yaml:"name"`
	Logo       string `json:"logo,omitempty" yaml:"logo,omitempty"`
	Color      string `json:"color,omitempty" yaml:"color,omitempty"`
	Background string `json:"background,omitempty" yaml:"background,omitempty"`
	Website    string `json:"website,omitempty" yaml:"website,omitempty"`
}

//go:embed all:vendor-data
var vendorData embed.FS

var vendorOnce sync.Once
var vendors []Vendor

// Vendors returns vendors sorted by display name.
func Vendors() []Vendor {
	vendorOnce.Do(func() {
		if deviceDir := diskDir(); deviceDir != "" {
			if loaded, err := loadVendors(os.DirFS(filepath.Join(filepath.Dir(deviceDir), "vendors"))); err == nil {
				vendors = loaded
				return
			}
		}
		if sub, err := fs.Sub(vendorData, "vendor-data"); err == nil {
			vendors, _ = loadVendors(sub)
		}
	})
	if vendors == nil {
		return []Vendor{}
	}
	return vendors
}

func loadVendors(fsys fs.FS) ([]Vendor, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Vendor
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, err := fs.ReadFile(fsys, entry.Name()+"/vendor.yaml")
		if err != nil {
			continue
		}
		var vendor Vendor
		if err := yaml.Unmarshal(raw, &vendor); err != nil {
			return nil, err
		}
		vendor.ID = entry.Name()
		if vendor.Logo != "" {
			vendor.Logo = "/api/vendors/" + vendor.ID + "/" + filepath.Base(vendor.Logo)
		}
		out = append(out, vendor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// VendorAsset reads a vendor logo from disk or the embedded vendor tree.
func VendorAsset(vendor, name string) ([]byte, error) {
	if filepath.Base(vendor) != vendor || filepath.Base(name) != name {
		return nil, fs.ErrNotExist
	}
	path := vendor + "/" + name
	if deviceDir := diskDir(); deviceDir != "" {
		if raw, err := os.ReadFile(filepath.Join(filepath.Dir(deviceDir), "vendors", path)); err == nil {
			return raw, nil
		}
	}
	sub, err := fs.Sub(vendorData, "vendor-data")
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(sub, path)
}
