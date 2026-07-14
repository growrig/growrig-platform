package catalogsource

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Fetch limits. Catalogs are YAML plus a few product images; anything near
// these caps is not a catalog package.
const (
	maxArchiveBytes = 200 << 20 // compressed download
	maxFileBytes    = 20 << 20  // single extracted file
	maxTotalBytes   = 500 << 20 // extracted tree
	fetchTimeout    = 60 * time.Second
)

var httpClient = &http.Client{Timeout: fetchTimeout}

// fetch downloads and extracts a provider-generated archive. The archive may
// contain catalog.yaml at its root or inside one wrapper directory, as is
// customary for Git hosting download endpoints.
func (m *Manager) fetch(archiveURL string) (Manifest, string, error) {
	if err := os.MkdirAll(m.cacheDir, 0o755); err != nil {
		return Manifest{}, "", err
	}
	work, err := os.MkdirTemp(m.cacheDir, ".fetch-*")
	if err != nil {
		return Manifest{}, "", err
	}
	ready := work + "-ready"
	cleanup := func() {
		_ = os.RemoveAll(work)
		_ = os.RemoveAll(ready)
	}

	archivePath := filepath.Join(work, "catalog.archive")
	if err := downloadArchive(archiveURL, archivePath); err != nil {
		cleanup()
		return Manifest{}, "", err
	}
	tree := filepath.Join(work, "tree")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		cleanup()
		return Manifest{}, "", err
	}
	if err := extractArchive(archivePath, tree); err != nil {
		cleanup()
		return Manifest{}, "", fmt.Errorf("extract catalog archive: %w", err)
	}
	root, err := findCatalogRoot(tree)
	if err != nil {
		cleanup()
		return Manifest{}, "", err
	}
	man, err := readManifest(root)
	if err != nil {
		cleanup()
		return Manifest{}, "", err
	}
	if err := os.Rename(root, ready); err != nil {
		cleanup()
		return Manifest{}, "", err
	}
	_ = os.RemoveAll(work)
	return man, ready, nil
}

func downloadArchive(archiveURL, path string) error {
	resp, err := httpClient.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download catalog archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download catalog archive: server returned %s (is the URL public and still valid?)", resp.Status)
	}
	if resp.ContentLength > maxArchiveBytes {
		return fmt.Errorf("catalog archive exceeds the %d MB download limit", maxArchiveBytes>>20)
	}
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(resp.Body, maxArchiveBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		return fmt.Errorf("download catalog archive: %w", copyErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxArchiveBytes {
		return fmt.Errorf("catalog archive exceeds the %d MB download limit", maxArchiveBytes>>20)
	}
	return nil
}

func extractArchive(path, dst string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	header := make([]byte, 512)
	n, err := io.ReadFull(f, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return err
	}
	header = header[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	switch {
	case len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b:
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		return extractTar(tar.NewReader(gz), dst)
	case len(header) >= 4 && string(header[:4]) == "PK\x03\x04":
		zr, err := zip.NewReader(f, info.Size())
		if err != nil {
			return err
		}
		return extractZip(zr, dst)
	case len(header) >= 262 && string(header[257:262]) == "ustar":
		return extractTar(tar.NewReader(f), dst)
	default:
		return fmt.Errorf("unsupported package format; expected .tar.gz, .tgz, .tar, or .zip")
	}
}

func extractTar(reader *tar.Reader, dst string) error {
	var total int64
	for {
		hdr, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		path, err := safeArchivePath(dst, hdr.Name)
		if err != nil {
			return err
		}
		if path == "" {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := extractFile(path, reader, hdr.Size, &total); err != nil {
				return err
			}
		case tar.TypeXHeader, tar.TypeXGlobalHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			continue
		default:
			return fmt.Errorf("archive entry %q has unsupported type", hdr.Name)
		}
	}
}

func extractZip(reader *zip.Reader, dst string) error {
	var total int64
	for _, entry := range reader.File {
		path, err := safeArchivePath(dst, entry.Name)
		if err != nil {
			return err
		}
		if path == "" {
			continue
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			return fmt.Errorf("archive entry %q has unsupported type", entry.Name)
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		err = extractFile(path, in, int64(entry.UncompressedSize64), &total)
		closeErr := in.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractFile(path string, reader io.Reader, size int64, total *int64) error {
	if size < 0 || size > maxFileBytes {
		return fmt.Errorf("file %s exceeds the %d MB limit", filepath.Base(path), maxFileBytes>>20)
	}
	*total += size
	if *total > maxTotalBytes {
		return fmt.Errorf("catalog exceeds the %d MB extracted-size limit", maxTotalBytes>>20)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(out, io.LimitReader(reader, maxFileBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != size {
		return fmt.Errorf("file %s has an invalid extracted size", filepath.Base(path))
	}
	return nil
}

func safeArchivePath(dst, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(name, "./")))
	if clean == "." || clean == "" {
		return "", nil
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("archive entry %q escapes the extraction root", name)
	}
	return filepath.Join(dst, clean), nil
}

func findCatalogRoot(tree string) (string, error) {
	var roots []string
	err := filepath.WalkDir(tree, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "catalog.yaml" {
			roots = append(roots, filepath.Dir(path))
			if len(roots) > 1 {
				return fmt.Errorf("archive contains more than one catalog.yaml")
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("not a GrowRig catalog: no catalog.yaml in the archive")
	}
	return roots[0], nil
}

func readManifest(dir string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "catalog.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, fmt.Errorf("not a GrowRig catalog: no catalog.yaml at the package root")
		}
		return Manifest{}, err
	}
	var man Manifest
	if err := yaml.Unmarshal(raw, &man); err != nil {
		return Manifest{}, fmt.Errorf("parse catalog.yaml: %w", err)
	}
	if err := man.validate(); err != nil {
		return Manifest{}, fmt.Errorf("invalid catalog.yaml: %w", err)
	}
	for _, kind := range man.Provides {
		if fi, err := os.Stat(filepath.Join(dir, kind)); err != nil || !fi.IsDir() {
			return Manifest{}, fmt.Errorf("invalid catalog: manifest provides %q but the package has no %s/ directory", kind, kind)
		}
	}
	if err := validateCatalogContent(dir, man.Provides); err != nil {
		return Manifest{}, fmt.Errorf("invalid catalog: %w", err)
	}
	return man, nil
}
