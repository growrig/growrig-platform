package catalogsource

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "devices"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `manifest: 1
id: test-catalog
name: Test Catalog
provides: [devices]
`
	if err := os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "test-catalog" || len(got.Provides) != 1 || got.Provides[0] != "devices" {
		t.Fatalf("manifest = %#v", got)
	}

	bad := `manifest: 1
id: ../escape
name: Bad
provides: [devices]
`
	if err := os.WriteFile(filepath.Join(dir, "catalog.yaml"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readManifest(dir); err == nil {
		t.Fatal("expected invalid manifest id to be rejected")
	}
}

func TestResolveRepositoryProviders(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		ref        string
		provider   string
		archive    string
	}{
		{"github", "https://github.com/owner/catalog.git", "main", "github", "https://codeload.github.com/owner/catalog/tar.gz/main"},
		{"gitlab", "https://gitlab.com/group/subgroup/catalog", "release/v1", "gitlab", "https://gitlab.com/api/v4/projects/group%2Fsubgroup%2Fcatalog/repository/archive.tar.gz?sha=release%2Fv1"},
		{"gitlab default", "https://gitlab.com/owner/catalog", "", "gitlab", "https://gitlab.com/api/v4/projects/owner%2Fcatalog/repository/archive.tar.gz"},
		{"bitbucket", "https://bitbucket.org/workspace/catalog", "main", "bitbucket", "https://bitbucket.org/workspace/catalog/get/main.gz"},
		{"codeberg", "https://codeberg.org/owner/catalog", "v1", "codeberg", "https://codeberg.org/api/v1/repos/owner/catalog/archive/v1.tar.gz"},
		{"gitea", "https://gitea.com/owner/catalog", "", "gitea", "https://gitea.com/api/v1/repos/owner/catalog/archive/HEAD.tar.gz"},
		{"forgejo", "https://forge.example.test/owner/catalog", "main", "forgejo", "https://forge.example.test/api/v1/repos/owner/catalog/archive/main.tar.gz"},
		{"self-hosted gitea", "https://git.example.test/owner/catalog", "v2", "forgejo", "https://git.example.test/api/v1/repos/owner/catalog/archive/v2.tar.gz"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveRepository(test.repository, test.ref)
			if err != nil {
				t.Fatal(err)
			}
			if got.Provider != test.provider || got.ArchiveURL != test.archive {
				t.Fatalf("resolveRepository() = %#v", got)
			}
		})
	}
	for _, invalid := range []string{
		"owner/catalog",
		"ftp://github.com/owner/catalog",
		"https://user:secret@github.com/owner/catalog",
		"https://forge.example.test/group/subgroup/catalog",
	} {
		if _, err := resolveRepository(invalid, ""); err == nil {
			t.Errorf("resolveRepository(%q) succeeded", invalid)
		}
	}
}

func TestExtractArchiveFormatsAndRejectsTraversal(t *testing.T) {
	entries := []tarEntry{
		{name: "repo-main/catalog.yaml", body: "manifest: 1\n"},
		{name: "repo-main/devices/fan/example/device.yaml", body: "brand: Example\n"},
	}
	formats := map[string][]byte{
		"tar.gz": tarArchive(t, entries, true),
		"tar":    tarArchive(t, entries, false),
		"zip":    zipArchive(t, entries),
	}
	for name, archive := range formats {
		t.Run(name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "catalog."+name)
			if err := os.WriteFile(archivePath, archive, 0o600); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := extractArchive(archivePath, dir); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "repo-main", "devices", "fan", "example", "device.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(raw) != "brand: Example\n" {
				t.Fatalf("extracted content = %q", raw)
			}
		})
	}

	escape := tarArchive(t, []tarEntry{{name: "repo-main/../../outside", body: "nope"}}, true)
	escapePath := filepath.Join(t.TempDir(), "escape.tar.gz")
	if err := os.WriteFile(escapePath, escape, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(escapePath, t.TempDir()); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

func TestNewLoadsValidatedSourcesAndDirs(t *testing.T) {
	dir := t.TempDir()
	source := Source{
		ID:         "community",
		Repository: "https://gitlab.com/growrig/community-catalog",
		Provider:   "gitlab",
		ArchiveURL: "https://gitlab.com/api/v4/projects/growrig%2Fcommunity-catalog/repository/archive.tar.gz",
		Name:       "Community",
		Provides:   []string{"devices", "species"},
		AddedAt:    time.Now().UTC(),
		FetchedAt:  time.Now().UTC(),
	}
	m := &Manager{
		file:     filepath.Join(dir, "catalog-sources.yaml"),
		cacheDir: filepath.Join(dir, "catalog-cache"),
		sources:  []Source{source},
	}
	if err := m.save(); err != nil {
		t.Fatal(err)
	}
	deviceDir := filepath.Join(m.cacheDir, source.ID, "devices")
	if err := os.MkdirAll(deviceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	loaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	dirs := loaded.Dirs("devices")
	if len(dirs) != 1 || dirs[0].SourceID != source.ID || dirs[0].Dir != deviceDir {
		t.Fatalf("device dirs = %#v", dirs)
	}
	if dirs := loaded.Dirs("../devices"); dirs != nil {
		t.Fatalf("invalid kind returned dirs: %#v", dirs)
	}

	loaded.Apply = func() { t.Fatal("apply called for missing source") }
	if err := loaded.Remove("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Remove(missing) = %v", err)
	}
}

func TestNewMigratesLegacyGitHubSource(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{
		file:     filepath.Join(dir, "catalog-sources.yaml"),
		cacheDir: filepath.Join(dir, "catalog-cache"),
		sources: []Source{{
			ID: "legacy", Repo: "growrig/community-catalog", Ref: "main", Name: "Legacy",
			Provides: []string{"devices"}, AddedAt: time.Now().UTC(), FetchedAt: time.Now().UTC(),
		}},
	}
	if err := m.save(); err != nil {
		t.Fatal(err)
	}
	loaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	source := loaded.List()[0]
	if source.Repository != "https://github.com/growrig/community-catalog" || source.Provider != "github" || source.ArchiveURL != "https://codeload.github.com/growrig/community-catalog/tar.gz/main" || source.Repo != "" || source.Ref != "main" {
		t.Fatalf("migrated source = %#v", source)
	}
}

func TestAddPersistsAppliesAndRemovesSource(t *testing.T) {
	archive := tarArchive(t, []tarEntry{
		{name: "community-main/catalog.yaml", body: "manifest: 1\nid: community\nname: Community Catalog\nprovides: [devices]\n"},
		{name: "community-main/devices/sensor/example/device.yaml", body: "brand: Example\nmodel: Example Sensor\nconnection: wifi\n"},
	}, true)
	repository := "https://gitlab.com/growrig/community"
	archiveURL := "https://gitlab.com/api/v4/projects/growrig%2Fcommunity/repository/archive.tar.gz?sha=main"
	previousClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != archiveURL {
			t.Fatalf("download URL = %s", request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(archive)),
			Request:    request,
		}, nil
	})}
	t.Cleanup(func() { httpClient = previousClient })

	dir := t.TempDir()
	m, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	applied := 0
	m.Apply = func() { applied++ }
	source, err := m.Add(repository, "main")
	if err != nil {
		t.Fatal(err)
	}
	if source.ID != "community" || source.Repository != repository || source.Provider != "gitlab" || source.Ref != "main" || applied != 1 {
		t.Fatalf("source = %#v, applied = %d", source, applied)
	}
	if len(m.Dirs("devices")) != 1 {
		t.Fatalf("device dirs = %#v", m.Dirs("devices"))
	}
	if _, err := New(dir); err != nil {
		t.Fatalf("reload persisted sources: %v", err)
	}
	if err := m.Remove(source.ID); err != nil {
		t.Fatal(err)
	}
	if applied != 2 || len(m.List()) != 0 || len(m.Dirs("devices")) != 0 {
		t.Fatalf("source was not fully removed: applied=%d list=%#v dirs=%#v", applied, m.List(), m.Dirs("devices"))
	}
}

type tarEntry struct {
	name string
	body string
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func tarArchive(t *testing.T, entries []tarEntry, compressed bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	var writer io.Writer = &buf
	var gz *gzip.Writer
	if compressed {
		gz = gzip.NewWriter(&buf)
		writer = gz
	}
	tw := tar.NewWriter(writer)
	for _, entry := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if gz != nil {
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func zipArchive(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, entry := range entries {
		writer, err := zw.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(entry.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
