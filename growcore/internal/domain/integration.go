package domain

import "time"

// IntegrationInstance is a configured copy of an integration bundle. Config
// only contains non-secret values; credentials are encrypted separately and
// never returned by the API.
type IntegrationInstance struct {
	ID            string            `json:"id"`
	BundleID      string            `json:"bundleId"`
	Name          string            `json:"name"`
	Config        map[string]string `json:"config"`
	SecretFields  []string          `json:"secretFields,omitempty"`
	Enabled       bool              `json:"enabled"`
	Status        string            `json:"status"`
	StatusMessage string            `json:"statusMessage,omitempty"`
	LastCheckedAt *time.Time        `json:"lastCheckedAt,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// IntegrationBinding selects an instance for a capability. GrowID is empty
// for a global feature binding; a grow-specific row overrides it.
type IntegrationBinding struct {
	ID         string    `json:"id"`
	Feature    string    `json:"feature"`
	GrowID     string    `json:"growId,omitempty"`
	Capability string    `json:"capability"`
	InstanceID string    `json:"instanceId"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
