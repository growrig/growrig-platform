// Package config loads Grow Core's YAML configuration.
//
// Configuration is infrastructure only: how Grow Core listens, where it stores
// data, and which adapter it uses to reach devices. The grow-domain model —
// environments, devices, channel roles and Home Assistant entity bindings — is
// owned by Grow Core and persisted as portable YAML beside the runtime database.
// SQLite is reserved for runtime cache, cycles and historical readings.
//
// The same binary runs in two modes, differing only by configuration:
//
//   - Home Assistant OS add-on (default): talks to HA through the Supervisor
//     proxy (see growcore.yaml).
//   - Local development: connects to a remote Home Assistant using a long-lived
//     access token (see growcore.dev.yaml).
//
// A built-in simulator mode needs no Home Assistant at all and is the
// zero-config fallback when no config file is present (see growcore.sim.yaml).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// AdapterType selects how Grow Core reaches physical devices.
type AdapterType string

const (
	AdapterSimulator     AdapterType = "simulator"
	AdapterHomeAssistant AdapterType = "homeassistant"
)

type Config struct {
	Server        Server        `yaml:"server"`
	Storage       Storage       `yaml:"storage"`
	Control       Control       `yaml:"control"`
	Adapter       Adapter       `yaml:"adapter"`
	HomeAssistant HomeAssistant `yaml:"homeassistant"`
}

type Server struct {
	Addr string `yaml:"addr"`
	// WorkDir is the process working directory. Relative paths in this config
	// (storage.path, storage.dataDir, …) and runtime lookups that walk from the
	// working directory resolve from here. Empty keeps the process cwd.
	WorkDir string `yaml:"workDir"`
}

type Storage struct {
	Path string `yaml:"path"`
	// DataDir is the root for on-disk data written alongside the database:
	// camera archives (environments/), grow media (grows/), integration secrets,
	// catalog sources and preferences. Empty means "the directory containing the
	// database file", so existing installs keep their current paths.
	DataDir string `yaml:"dataDir"`
}

// DataDir is the resolved root directory for on-disk data. It is the configured
// storage.dataDir, or the directory containing the database when unset.
func (c *Config) DataDir() string {
	if c.Storage.DataDir != "" {
		return c.Storage.DataDir
	}
	return filepath.Dir(c.Storage.Path)
}

// EnvironmentsDir holds per-environment media (camera archives).
func (c *Config) EnvironmentsDir() string { return filepath.Join(c.DataDir(), "environments") }

// GrowsDir holds per-grow media (photos).
func (c *Config) GrowsDir() string { return filepath.Join(c.DataDir(), "grows") }

type Control struct {
	Interval Duration `yaml:"interval"`
}

type Adapter struct {
	Type AdapterType `yaml:"type"`
}

type HomeAssistant struct {
	// URL is the base URL of Home Assistant, e.g. http://homeassistant.local:8123
	// or, in a HAOS add-on, the Supervisor core proxy http://supervisor/core.
	URL string `yaml:"url"`
	// Token is a long-lived access token (dev) or the Supervisor token
	// (add-on). Supports ${ENV_VAR} expansion so secrets stay out of the file.
	Token string `yaml:"token"`
}

// Default returns the zero-config simulator setup so `growcore` runs with no
// config file and no hardware.
func Default() *Config {
	return &Config{
		Server:  Server{Addr: ":8080"},
		Storage: Storage{Path: "growcore.db"},
		Control: Control{Interval: Duration(2 * time.Second)},
		Adapter: Adapter{Type: AdapterSimulator},
	}
}

// Load reads and validates configuration from path. Environment references of
// the form ${VAR} in the file are expanded before parsing. Missing scalar
// settings fall back to the simulator defaults.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	expanded := os.Expand(string(raw), os.Getenv)

	cfg := Default()
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	switch c.Adapter.Type {
	case AdapterSimulator:
	case AdapterHomeAssistant:
		if c.HomeAssistant.URL == "" {
			return fmt.Errorf("homeassistant.url is required when adapter.type is homeassistant")
		}
		if c.HomeAssistant.Token == "" {
			return fmt.Errorf("homeassistant.token is required when adapter.type is homeassistant (set the referenced environment variable)")
		}
	default:
		return fmt.Errorf("unknown adapter.type %q", c.Adapter.Type)
	}
	return nil
}

// ApplyWorkDir changes the process working directory to server.workDir when set.
// Relative config paths (storage, data) and Getwd-based lookups then resolve
// from that directory. No-op when workDir is empty.
func (c *Config) ApplyWorkDir() error {
	if c.Server.WorkDir == "" {
		return nil
	}
	if err := os.Chdir(c.Server.WorkDir); err != nil {
		return fmt.Errorf("server.workDir %q: %w", c.Server.WorkDir, err)
	}
	return nil
}

// Duration is a time.Duration that unmarshals from a YAML string like "2s".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }
