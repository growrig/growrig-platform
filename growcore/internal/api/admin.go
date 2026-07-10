package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/growrig/growrig-platform/growcore/internal/domain"
)

// --- Environments ---

type environmentBody struct {
	Name           string                 `json:"name"`
	Kind           domain.EnvironmentKind `json:"kind"`
	AirSourceID    string                 `json:"airSourceId"`
	Model          string                 `json:"model"`
	WidthCm        float64                `json:"widthCm"`
	DepthCm        float64                `json:"depthCm"`
	HeightCm       float64                `json:"heightCm"`
	TargetTempC    float64                `json:"targetTempC"`
	TargetHumidity float64                `json:"targetHumidity"`
	TargetCO2      float64                `json:"targetCO2"`
	EmergencyTempC float64                `json:"emergencyTempC"`
}

func (s *Server) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var b environmentBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	env, err := s.buildEnvironment(id(b.Name, "env"), b)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	if err := s.store.SaveEnvironment(env); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	var b environmentBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	env, err := s.buildEnvironment(r.PathValue("id"), b)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	if err := s.store.SaveEnvironment(env); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (s *Server) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteEnvironment(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusConflict, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) buildEnvironment(envID string, b environmentBody) (domain.Environment, error) {
	if strings.TrimSpace(b.Name) == "" {
		return domain.Environment{}, fmt.Errorf("name is required")
	}
	kind := b.Kind
	if kind == "" {
		kind = domain.KindTent
	}
	if kind != domain.KindTent && kind != domain.KindRoom {
		return domain.Environment{}, fmt.Errorf("unknown kind %q", kind)
	}
	env := domain.Environment{
		ID: envID, Name: b.Name, Kind: kind, AirSourceID: b.AirSourceID,
		Model:          b.Model,
		WidthCm:        b.WidthCm,
		DepthCm:        b.DepthCm,
		HeightCm:       b.HeightCm,
		TargetTempC:    orDefault(b.TargetTempC, 24),
		TargetHumidity: orDefault(b.TargetHumidity, 55),
		TargetCO2:      b.TargetCO2,
		EmergencyTempC: orDefault(b.EmergencyTempC, 35),
	}
	if kind == domain.KindRoom {
		env.AirSourceID = "" // rooms don't have an air source
	}
	if env.AirSourceID != "" && env.AirSourceID == envID {
		return domain.Environment{}, fmt.Errorf("an environment cannot be its own air source")
	}
	return env, nil
}

// --- Bindings ---

type bindingBody struct {
	EnvironmentID string             `json:"environmentId"`
	Kind          domain.BindingKind `json:"kind"`
	Name          string             `json:"name"`
	Entity        string             `json:"entity"`
	Measurement   domain.Measurement `json:"measurement"`
	Role          domain.Role        `json:"role"`
	RPMEntity     string             `json:"rpmEntity"`
	Wattage       float64            `json:"wattage"`
	Primary       bool               `json:"primary"`
}

func (s *Server) createBinding(w http.ResponseWriter, r *http.Request) {
	var b bindingBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	binding, err := s.buildBinding(id(b.Name, string(b.Kind)), b)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	if err := s.store.SaveBinding(binding); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if binding.Kind == domain.KindLight {
		if binding.Primary {
			if err := s.store.SetPrimaryLight(binding.EnvironmentID, binding.ID); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			binding.Primary = true
		} else if err := s.store.EnsurePrimaryLight(binding.EnvironmentID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, binding)
}

func (s *Server) updateBinding(w http.ResponseWriter, r *http.Request) {
	var b bindingBody
	if err := decode(r, &b); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	binding, err := s.buildBinding(r.PathValue("id"), b)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err.Error()))
		return
	}
	if err := s.store.SaveBinding(binding); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if binding.Kind == domain.KindLight {
		if binding.Primary {
			if err := s.store.SetPrimaryLight(binding.EnvironmentID, binding.ID); err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
		} else if err := s.store.EnsurePrimaryLight(binding.EnvironmentID); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, binding)
}

func (s *Server) deleteBinding(w http.ResponseWriter, r *http.Request) {
	bindingID := r.PathValue("id")

	// Remember the environment of a light so we can promote a new primary
	// after removal.
	var lightEnv string
	if all, err := s.store.Bindings(); err == nil {
		for _, b := range all {
			if b.ID == bindingID && b.Kind == domain.KindLight {
				lightEnv = b.EnvironmentID
				break
			}
		}
	}

	if err := s.store.DeleteBinding(bindingID); err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	if lightEnv != "" {
		if err := s.store.EnsurePrimaryLight(lightEnv); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) buildBinding(bindingID string, b bindingBody) (domain.Binding, error) {
	if strings.TrimSpace(b.Name) == "" {
		return domain.Binding{}, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(b.Entity) == "" {
		return domain.Binding{}, fmt.Errorf("entity is required")
	}
	envs, err := s.store.Environments()
	if err != nil {
		return domain.Binding{}, err
	}
	if !containsEnv(envs, b.EnvironmentID) {
		return domain.Binding{}, fmt.Errorf("unknown environment %q", b.EnvironmentID)
	}

	binding := domain.Binding{
		ID: bindingID, EnvironmentID: b.EnvironmentID, Kind: b.Kind, Name: b.Name, Entity: b.Entity,
	}
	switch b.Kind {
	case domain.KindSensor:
		switch b.Measurement {
		case domain.MeasureTemperature, domain.MeasureHumidity, domain.MeasureCO2:
			binding.Measurement = b.Measurement
		default:
			return domain.Binding{}, fmt.Errorf("sensor needs a measurement (temperature, humidity or co2)")
		}
	case domain.KindFan:
		role := b.Role
		if role == "" {
			role = domain.RoleUnassigned
		}
		if !validRole(role) {
			return domain.Binding{}, fmt.Errorf("unknown role %q", role)
		}
		binding.Role = role
		binding.RPMEntity = b.RPMEntity
	case domain.KindLight:
		if b.Wattage < 0 || b.Wattage > 100000 {
			return domain.Binding{}, fmt.Errorf("wattage must be between 0 and 100000")
		}
		binding.Wattage = b.Wattage
		binding.Primary = b.Primary
	case domain.KindCamera:
		// no extra fields
	default:
		return domain.Binding{}, fmt.Errorf("unknown binding kind %q", b.Kind)
	}
	return binding, nil
}

// --- helpers ---

func containsEnv(envs []domain.Environment, id string) bool {
	for _, e := range envs {
		if e.ID == id {
			return true
		}
	}
	return false
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func errBody(msg string) map[string]string { return map[string]string{"error": msg} }

func orDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// id builds a readable, unique id from a name plus a short random suffix.
func id(name, prefix string) string {
	slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" {
		slug = prefix
	}
	return slug + "-" + randHex(3)
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
