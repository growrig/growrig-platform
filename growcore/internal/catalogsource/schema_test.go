package catalogsource

import (
	"os"
	"path/filepath"
	"testing"
)

// repoCatalogDir walks up to the repo-root catalog/ submodule, or skips.
func repoCatalogDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "catalog")
		if _, err := os.Stat(filepath.Join(cand, "catalog.yaml")); err == nil {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("catalog submodule not checked out")
	return ""
}

// The shipped default catalog must satisfy the published schemas — this guards
// against a schema drifting stricter than the real content it describes.
func TestDefaultCatalogMatchesSchemas(t *testing.T) {
	dir := repoCatalogDir(t)
	if err := validateCatalogContent(dir, Kinds); err != nil {
		t.Fatalf("default catalog failed schema validation: %v", err)
	}
}

func TestValidateRejectsBadContent(t *testing.T) {
	if set := loadSchemas(); set.err != nil {
		t.Skipf("schemas unavailable: %v", set.err)
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "catalog.yaml"),
		"manifest: 1\nid: bad\nname: Bad\nprovides: [devices]\n")

	// A device missing the required `connection`, plus an unknown key.
	dev := filepath.Join(root, "devices", "fan", "acme-fan")
	if err := os.MkdirAll(dev, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dev, "device.yaml"),
		"brand: Acme\nmodel: Blower\nwattage: 12\n") // wattage is not a device field

	err := validateCatalogContent(root, []string{"devices"})
	if err == nil {
		t.Fatal("expected validation error for malformed device.yaml, got nil")
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
