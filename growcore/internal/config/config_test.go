package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "growcore.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadExpandsEnvAndParsesDuration(t *testing.T) {
	t.Setenv("TEST_HA_TOKEN", "secret-token")
	p := writeTemp(t, `
server:
  addr: ":9000"
control:
  interval: 3s
adapter:
  type: homeassistant
homeassistant:
  url: http://homeassistant.local:8123
  token: ${TEST_HA_TOKEN}
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HomeAssistant.Token != "secret-token" {
		t.Errorf("token not expanded: %q", cfg.HomeAssistant.Token)
	}
	if cfg.Control.Interval.Std().String() != "3s" {
		t.Errorf("interval = %s, want 3s", cfg.Control.Interval.Std())
	}
	if cfg.Server.Addr != ":9000" {
		t.Errorf("addr = %s", cfg.Server.Addr)
	}
}

func TestPartialConfigKeepsDefaults(t *testing.T) {
	p := writeTemp(t, "adapter:\n  type: simulator\n")
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("addr default lost: %q", cfg.Server.Addr)
	}
	if cfg.Storage.Path != "growcore.db" {
		t.Errorf("storage default lost: %q", cfg.Storage.Path)
	}
}

func TestValidateRejectsHAWithoutToken(t *testing.T) {
	p := writeTemp(t, `
adapter:
  type: homeassistant
homeassistant:
  url: http://homeassistant.local:8123
`)
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestLoadWorkDir(t *testing.T) {
	p := writeTemp(t, `
server:
  addr: ":8080"
  workDir: /tmp/growcore-data
adapter:
  type: simulator
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.WorkDir != "/tmp/growcore-data" {
		t.Errorf("workDir = %q", cfg.Server.WorkDir)
	}
}

func TestApplyWorkDir(t *testing.T) {
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(before) })

	dir := t.TempDir()
	cfg := Default()
	cfg.Server.WorkDir = dir
	if err := cfg.ApplyWorkDir(); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	got, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("cwd = %q, want %q", got, want)
	}
}

func TestApplyWorkDirEmptyIsNoop(t *testing.T) {
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := cfg.ApplyWorkDir(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Errorf("cwd changed from %q to %q", before, after)
	}
}
