package integrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
	"github.com/growrig/growrig-platform/growcore/internal/store"
)

type Manager struct {
	store   *store.Store
	root    string
	bundles map[string]Bundle
	vault   *vault
}

type InstanceInput struct {
	BundleID string            `json:"bundleId"`
	Name     string            `json:"name"`
	Config   map[string]string `json:"config"`
	Enabled  *bool             `json:"enabled,omitempty"`
}
type BindingInput struct {
	Feature    string `json:"feature"`
	GrowID     string `json:"growId"`
	Capability string `json:"capability"`
	InstanceID string `json:"instanceId"`
}

func NewManager(st *store.Store, root, keyPath string) (*Manager, error) {
	if root == "" {
		root = FindBundleRoot()
	}
	bs, err := LoadBundles(root)
	if err != nil {
		return nil, err
	}
	if len(bs) == 0 {
		bs, err = loadEmbeddedBundles()
		if err != nil {
			return nil, err
		}
	}
	v, err := openVault(keyPath)
	if err != nil {
		return nil, err
	}
	m := &Manager{store: st, root: root, bundles: map[string]Bundle{}, vault: v}
	for _, b := range bs {
		m.bundles[b.ID] = b
	}
	if err := m.ensureDefaultIntegrations(); err != nil {
		return nil, err
	}
	if err := m.ensureDefaultAIChatBinding(); err != nil {
		return nil, err
	}
	return m, nil
}

// ensureDefaultIntegrations seeds services GrowRig already depends on. It is
// idempotent and never replaces a user's existing Open-Meteo instance or
// weather binding.
func (m *Manager) ensureDefaultIntegrations() error {
	if _, available := m.bundles["open-meteo"]; !available {
		return nil
	}
	records, err := m.store.IntegrationInstances()
	if err != nil {
		return err
	}
	var instanceID string
	for _, record := range records {
		if record.Instance.BundleID == "open-meteo" {
			instanceID = record.Instance.ID
			break
		}
	}
	if instanceID == "" {
		instance, err := m.Create(InstanceInput{BundleID: "open-meteo", Name: "Open-Meteo", Config: map[string]string{}})
		if err != nil {
			return fmt.Errorf("create default Open-Meteo integration: %w", err)
		}
		instanceID = instance.ID
	}
	bindings, err := m.store.IntegrationBindings()
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.Feature == "weather-context" && binding.GrowID == "" && binding.Capability == "weather.forecast" {
			return nil
		}
	}
	now := time.Now()
	binding := domain.IntegrationBinding{ID: newID("ib"), Feature: "weather-context", Capability: "weather.forecast", InstanceID: instanceID, CreatedAt: now, UpdatedAt: now}
	if err := m.store.SaveIntegrationBinding(binding); err != nil {
		return fmt.Errorf("bind default Open-Meteo integration: %w", err)
	}
	return nil
}

// ensureDefaultAIChatBinding makes the first enabled chat-capable instance
// immediately useful. Explicit existing bindings always win, including
// grow-specific overrides configured later.
func (m *Manager) ensureDefaultAIChatBinding() error {
	bindings, err := m.store.IntegrationBindings()
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.Feature == "grow-assistant" && binding.GrowID == "" && binding.Capability == "ai.chat" {
			return nil
		}
	}
	records, err := m.store.IntegrationInstances()
	if err != nil {
		return err
	}
	for _, record := range records {
		bundle, ok := m.bundles[record.Instance.BundleID]
		if !ok || !record.Instance.Enabled || !bundle.hasCapability("ai.chat") {
			continue
		}
		now := time.Now()
		return m.store.SaveIntegrationBinding(domain.IntegrationBinding{
			ID: newID("ib"), Feature: "grow-assistant", Capability: "ai.chat",
			InstanceID: record.Instance.ID, CreatedAt: now, UpdatedAt: now,
		})
	}
	return nil
}

func (m *Manager) Bundles() []Bundle {
	out := make([]Bundle, 0, len(m.bundles))
	for _, b := range m.bundles {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].Name < out[j].Name
		}
		return out[i].Category < out[j].Category
	})
	return out
}
func (m *Manager) BundleAsset(id, name string) ([]byte, error) {
	b, ok := m.bundles[id]
	if !ok {
		return nil, os.ErrNotExist
	}
	if name != "icon.svg" && name != "README.md" {
		return nil, os.ErrNotExist
	}
	if b.assetFS != nil {
		return fs.ReadFile(b.assetFS, b.assetRoot+"/"+name)
	}
	return os.ReadFile(filepath.Join(b.dir, name))
}

func (m *Manager) Instances() ([]domain.IntegrationInstance, error) {
	recs, err := m.store.IntegrationInstances()
	if err != nil {
		return nil, err
	}
	out := make([]domain.IntegrationInstance, 0, len(recs))
	for _, r := range recs {
		r.Instance.SecretFields = m.secretFieldNames(r.Instance.BundleID, r.Secrets)
		out = append(out, r.Instance)
	}
	return out, nil
}

func (m *Manager) Create(in InstanceInput) (domain.IntegrationInstance, error) {
	b, ok := m.bundles[in.BundleID]
	if !ok {
		return domain.IntegrationInstance{}, fmt.Errorf("unknown integration bundle %q", in.BundleID)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = b.Name
	}
	pub, secrets, err := m.validateConfig(b, in.Config, nil)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	sealed, err := m.vault.encrypt(secrets)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	now := time.Now()
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	inst := domain.IntegrationInstance{ID: newID("int"), BundleID: b.ID, Name: name, Config: pub, Enabled: enabled, Status: "unknown", CreatedAt: now, UpdatedAt: now}
	if err := m.store.SaveIntegrationInstance(store.IntegrationRecord{Instance: inst, Secrets: sealed}); err != nil {
		return domain.IntegrationInstance{}, err
	}
	if b.hasCapability("ai.chat") {
		if err := m.ensureDefaultAIChatBinding(); err != nil {
			return domain.IntegrationInstance{}, err
		}
	}
	inst.SecretFields = m.secretFieldNames(b.ID, sealed)
	return inst, nil
}

func (m *Manager) Update(id string, in InstanceInput) (domain.IntegrationInstance, error) {
	rec, err := m.store.IntegrationInstance(id)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	b, ok := m.bundles[rec.Instance.BundleID]
	if !ok {
		return domain.IntegrationInstance{}, fmt.Errorf("integration bundle %q is unavailable", rec.Instance.BundleID)
	}
	oldSecrets, err := m.vault.decrypt(rec.Secrets)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	merged := map[string]string{}
	for k, v := range rec.Instance.Config {
		merged[k] = v
	}
	for k, v := range in.Config {
		if v != "" {
			merged[k] = v
		}
	}
	for k, v := range oldSecrets {
		if _, present := in.Config[k]; !present || in.Config[k] == "" {
			merged[k] = v
		}
	}
	pub, secrets, err := m.validateConfig(b, merged, oldSecrets)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	sealed, err := m.vault.encrypt(secrets)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	if strings.TrimSpace(in.Name) != "" {
		rec.Instance.Name = strings.TrimSpace(in.Name)
	}
	if in.Enabled != nil {
		rec.Instance.Enabled = *in.Enabled
		if !*in.Enabled {
			rec.Instance.Status = "disabled"
			rec.Instance.StatusMessage = ""
		} else if rec.Instance.Status == "disabled" {
			rec.Instance.Status = "unknown"
		}
	}
	rec.Instance.Config = pub
	rec.Instance.UpdatedAt = time.Now()
	rec.Secrets = sealed
	if err = m.store.SaveIntegrationInstance(rec); err != nil {
		return domain.IntegrationInstance{}, err
	}
	rec.Instance.SecretFields = m.secretFieldNames(b.ID, sealed)
	return rec.Instance, nil
}

func (m *Manager) Delete(id string) error { return m.store.DeleteIntegrationInstance(id) }

func (m *Manager) Test(ctx context.Context, id string) (domain.IntegrationInstance, error) {
	rec, b, cfg, err := m.runtimeConfig(id)
	if err != nil {
		return domain.IntegrationInstance{}, err
	}
	if !rec.Instance.Enabled {
		return domain.IntegrationInstance{}, errors.New("integration instance is disabled")
	}
	start := time.Now()
	err = runTest(ctx, b, cfg)
	now := time.Now()
	rec.Instance.LastCheckedAt = &now
	rec.Instance.UpdatedAt = now
	if err != nil {
		rec.Instance.Status = "error"
		rec.Instance.StatusMessage = err.Error()
	} else {
		rec.Instance.Status = "healthy"
		rec.Instance.StatusMessage = fmt.Sprintf("Connected in %d ms", time.Since(start).Milliseconds())
	}
	_ = m.store.SaveIntegrationInstance(rec)
	rec.Instance.SecretFields = m.secretFieldNames(b.ID, rec.Secrets)
	return rec.Instance, err
}

func (m *Manager) Invoke(ctx context.Context, id, capability string, input map[string]any) (any, error) {
	rec, b, cfg, err := m.runtimeConfig(id)
	if err != nil {
		return nil, err
	}
	if !rec.Instance.Enabled {
		return nil, errors.New("integration instance is disabled")
	}
	if !b.hasCapability(capability) {
		return nil, fmt.Errorf("integration does not provide %s", capability)
	}
	return runOperation(ctx, b, cfg, capability, input)
}

func (m *Manager) Bindings() ([]domain.IntegrationBinding, error) {
	return m.store.IntegrationBindings()
}
func (m *Manager) SaveBinding(in BindingInput) (domain.IntegrationBinding, error) {
	if strings.TrimSpace(in.Feature) == "" || strings.TrimSpace(in.Capability) == "" {
		return domain.IntegrationBinding{}, errors.New("feature and capability are required")
	}
	rec, b, _, err := m.runtimeConfig(in.InstanceID)
	if err != nil {
		return domain.IntegrationBinding{}, err
	}
	if !rec.Instance.Enabled {
		return domain.IntegrationBinding{}, errors.New("integration instance is disabled")
	}
	if !b.hasCapability(in.Capability) {
		return domain.IntegrationBinding{}, fmt.Errorf("%s does not provide %s", rec.Instance.Name, in.Capability)
	}
	now := time.Now()
	binding := domain.IntegrationBinding{ID: newID("ib"), Feature: in.Feature, GrowID: in.GrowID, Capability: in.Capability, InstanceID: in.InstanceID, CreatedAt: now, UpdatedAt: now}
	if err = m.store.SaveIntegrationBinding(binding); err != nil {
		return domain.IntegrationBinding{}, err
	}
	return binding, nil
}
func (m *Manager) DeleteBinding(id string) error { return m.store.DeleteIntegrationBinding(id) }
func (m *Manager) Resolve(feature, growID, capability string) (*domain.IntegrationInstance, error) {
	instances, err := m.Instances()
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.IntegrationInstance{}
	for _, i := range instances {
		byID[i.ID] = i
	}
	bindings, err := m.Bindings()
	if err != nil {
		return nil, err
	}
	for _, wantedGrow := range []string{growID, ""} {
		for _, b := range bindings {
			if b.Feature == feature && b.GrowID == wantedGrow && b.Capability == capability {
				if i, ok := byID[b.InstanceID]; ok && i.Enabled {
					return &i, nil
				}
			}
		}
	}
	return nil, nil
}

func (m *Manager) runtimeConfig(id string) (store.IntegrationRecord, Bundle, map[string]string, error) {
	rec, err := m.store.IntegrationInstance(id)
	if err != nil {
		return rec, Bundle{}, nil, err
	}
	b, ok := m.bundles[rec.Instance.BundleID]
	if !ok {
		return rec, Bundle{}, nil, fmt.Errorf("integration bundle %q is unavailable", rec.Instance.BundleID)
	}
	secrets, err := m.vault.decrypt(rec.Secrets)
	if err != nil {
		return rec, b, nil, err
	}
	cfg := map[string]string{}
	for k, v := range rec.Instance.Config {
		cfg[k] = v
	}
	for k, v := range secrets {
		cfg[k] = v
	}
	return rec, b, cfg, nil
}
func (m *Manager) validateConfig(b Bundle, input, existingSecrets map[string]string) (map[string]string, map[string]string, error) {
	pub := map[string]string{}
	secrets := map[string]string{}
	for _, f := range b.Config {
		v := strings.TrimSpace(input[f.Key])
		if v == "" {
			v = f.Default
		}
		if v == "" && f.Secret && existingSecrets != nil {
			v = existingSecrets[f.Key]
		}
		if f.Required && v == "" {
			return nil, nil, fmt.Errorf("%s is required", f.Label)
		}
		if f.Type == "url" && v != "" && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			return nil, nil, fmt.Errorf("%s must be an http or https URL", f.Label)
		}
		if len(f.Options) > 0 && v != "" {
			valid := false
			for _, o := range f.Options {
				if v == o {
					valid = true
				}
			}
			if !valid {
				return nil, nil, fmt.Errorf("invalid value for %s", f.Label)
			}
		}
		if f.Secret {
			if v != "" {
				secrets[f.Key] = v
			}
		} else {
			pub[f.Key] = v
		}
	}
	return pub, secrets, nil
}
func (m *Manager) secretFieldNames(bundleID, sealed string) []string {
	if sealed == "" {
		return nil
	}
	b, ok := m.bundles[bundleID]
	if !ok {
		return nil
	}
	out := []string{}
	for _, f := range b.Config {
		if f.Secret {
			out = append(out, f.Key)
		}
	}
	return out
}
func newID(prefix string) string {
	raw := make([]byte, 8)
	_, _ = rand.Read(raw)
	return prefix + "-" + hex.EncodeToString(raw)
}
