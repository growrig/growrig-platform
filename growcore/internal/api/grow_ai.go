package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

type growAIStatus struct {
	Available    bool   `json:"available"`
	InstanceName string `json:"instanceName,omitempty"`
}

func (s *Server) getAIStatus(w http.ResponseWriter, r *http.Request) {
	instance, err := s.integrations.Resolve("grow-assistant", "", "ai.chat")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if instance == nil {
		writeJSON(w, http.StatusOK, growAIStatus{})
		return
	}
	writeJSON(w, http.StatusOK, growAIStatus{Available: true, InstanceName: instance.Name})
}

func (s *Server) getGrowAIStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok, err := s.store.Grow(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	} else if !ok {
		writeJSON(w, http.StatusNotFound, errBody("grow not found"))
		return
	}
	instance, err := s.integrations.Resolve("grow-assistant", r.PathValue("id"), "ai.chat")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if instance == nil {
		writeJSON(w, http.StatusOK, growAIStatus{})
		return
	}
	writeJSON(w, http.StatusOK, growAIStatus{Available: true, InstanceName: instance.Name})
}

type growAIChatBody struct {
	ChatID        string `json:"chatId"`
	Content       string `json:"content"`
	GrowID        string `json:"growId"`
	EnvironmentID string `json:"environmentId"`
}

func (s *Server) chatWithGrowAI(w http.ResponseWriter, r *http.Request) {
	var body growAIChatBody
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	body.Content = strings.TrimSpace(body.Content)
	if body.Content == "" || len(body.Content) > 4000 {
		writeJSON(w, http.StatusBadRequest, errBody("message must contain 1–4000 characters"))
		return
	}

	growID, environmentID := body.GrowID, body.EnvironmentID
	pathGrowID := r.PathValue("id")
	if pathGrowID != "" {
		growID = pathGrowID
	}
	if growID != "" && environmentID != "" {
		writeJSON(w, http.StatusBadRequest, errBody("choose either a grow or an environment context"))
		return
	}
	user, _ := currentUser(r)
	var chat domain.AIChat
	var history []domain.AIChatMessage
	instanceID, instanceName := "", ""
	if body.ChatID != "" {
		var ok bool
		var err error
		chat, ok, err = s.store.AIChat(body.ChatID, user.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !ok || (pathGrowID != "" && chat.GrowID != pathGrowID) {
			writeJSON(w, http.StatusNotFound, errBody("chat not found"))
			return
		}
		growID, environmentID = chat.GrowID, chat.EnvironmentID
		if chat.Archived {
			writeJSON(w, http.StatusConflict, errBody("restore this chat before sending another message"))
			return
		}
		instanceID, instanceName = chat.InstanceID, chat.InstanceName
		history, err = s.store.AIChatMessages(chat.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		instance, err := s.integrations.ResolveFor("grow-assistant", growID, environmentID, "ai.chat")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if instance == nil {
			writeJSON(w, http.StatusConflict, errBody("no enabled ai.chat integration is bound to Grow assistant"))
			return
		}
		instanceID, instanceName = instance.ID, instance.Name
	}
	contextJSON, err := s.assistantAIContext(user, growID, environmentID)
	if errors.Is(err, errGrowNotFound) || errors.Is(err, errEnvironmentNotFound) {
		writeJSON(w, http.StatusNotFound, errBody(err.Error()))
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	messages := []map[string]string{{
		"role": "system",
		"content": `You are GrowRig's read-only assistant. Answer using the supplied GrowRig context and respect its selected scope. Use measurements and timestamps as evidence, distinguish facts from hypotheses, and say when context is insufficient. Never claim that you changed a device, automation, care log, or plant record. Any action must be phrased as a proposal requiring user review. Do not give pesticide, chemical, or safety-critical instructions without a caution to verify the product label and local guidance. Keep answers concise and practical.

Current GrowRig context (JSON):
` + string(contextJSON),
	}}
	if len(history) > 19 {
		history = history[len(history)-19:]
	}
	for _, message := range history {
		messages = append(messages, map[string]string{"role": message.Role, "content": message.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": body.Content})
	userMessageCreatedAt := time.Now()
	result, err := s.integrations.Invoke(r.Context(), instanceID, "ai.chat", map[string]any{"messages": messages, "stream": false})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	content, err := integrationChatContent(result)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	now := userMessageCreatedAt
	var create *domain.AIChat
	if chat.ID == "" {
		chat = domain.AIChat{
			ID: id(body.Content, "chat"), UserID: user.ID, GrowID: growID, EnvironmentID: environmentID,
			Title: chatTitle(body.Content), InstanceID: instanceID,
			CreatedAt: now, UpdatedAt: now,
		}
		create = &chat
	}
	userMessage := domain.AIChatMessage{ID: id("user", "msg"), ChatID: chat.ID, Role: "user", Content: body.Content, CreatedAt: now}
	assistantMessage := domain.AIChatMessage{ID: id("assistant", "msg"), ChatID: chat.ID, Role: "assistant", Content: content, CreatedAt: time.Now()}
	if err := s.store.SaveAIChatExchange(create, userMessage, assistantMessage); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	chat, _, _ = s.store.AIChat(chat.ID, user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"chat": chat, "message": assistantMessage, "instanceName": instanceName})
}

func chatTitle(content string) string {
	content = strings.Join(strings.Fields(content), " ")
	runes := []rune(content)
	if len(runes) > 60 {
		return string(runes[:57]) + "…"
	}
	return content
}

func (s *Server) getAIChats(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	var archived *bool
	if value := r.URL.Query().Get("archived"); value == "true" || value == "false" {
		wanted := value == "true"
		archived = &wanted
	}
	chats, err := s.store.AIChats(user.ID, archived)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, chats)
}

func (s *Server) getAIChat(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	chat, ok, err := s.store.AIChat(r.PathValue("id"), user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("chat not found"))
		return
	}
	chat.Messages, err = s.store.AIChatMessages(chat.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, chat)
}

type updateAIChatBody struct {
	Archived *bool `json:"archived"`
}

func (s *Server) updateAIChat(w http.ResponseWriter, r *http.Request) {
	var body updateAIChatBody
	if err := decode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if body.Archived == nil {
		writeJSON(w, http.StatusBadRequest, errBody("archived is required"))
		return
	}
	user, _ := currentUser(r)
	ok, err := s.store.SetAIChatArchived(r.PathValue("id"), user.ID, *body.Archived)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody("chat not found"))
		return
	}
	chat, _, err := s.store.AIChat(r.PathValue("id"), user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, chat)
}

var errGrowNotFound = errors.New("grow not found")
var errEnvironmentNotFound = errors.New("environment not found")

func (s *Server) assistantAIContext(user *domain.User, growID, environmentID string) ([]byte, error) {
	if growID != "" {
		context, err := s.growAIContext(growID)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"scope": "grow", "context": json.RawMessage(context)})
	}
	if environmentID != "" {
		allowed, all := s.accessibleEnvIDs(user)
		if !all && !allowed[environmentID] {
			return nil, errEnvironmentNotFound
		}
		environments, err := s.store.Environments()
		if err != nil {
			return nil, err
		}
		var selected *domain.Environment
		for i := range environments {
			if environments[i].ID == environmentID {
				selected = &environments[i]
				break
			}
		}
		if selected == nil {
			return nil, errEnvironmentNotFound
		}
		plants, err := s.store.PlantsInEnvironment(environmentID)
		if err != nil {
			return nil, err
		}
		growIDs := map[string]bool{}
		for _, plant := range plants {
			growIDs[plant.GrowID] = true
		}
		grows := []domain.Grow{}
		for growID := range growIDs {
			grow, ok, err := s.store.Grow(growID)
			if err != nil {
				return nil, err
			}
			if ok {
				grows = append(grows, grow)
			}
		}
		var live *domain.EnvironmentView
		snapshot := s.engine.Latest()
		for i := range snapshot.Environments {
			if snapshot.Environments[i].ID == environmentID {
				value := snapshot.Environments[i]
				live = &value
				break
			}
		}
		readings, err := s.store.ReadingsSince(environmentID, time.Now().Add(-7*24*time.Hour), 48)
		if err != nil {
			return nil, err
		}
		activities, err := s.store.Activities(environmentID, "", nil, nil, 25, 0)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{
			"scope": "environment", "generatedAt": time.Now(), "environment": selected,
			"currentState": live, "plants": plants, "grows": grows,
			"historyWindow":  "last 7 days, downsampled to at most 48 averaged readings",
			"climateHistory": readings, "recentActivity": activities,
		})
	}
	allowed, all := s.accessibleEnvIDs(user)
	snapshot := filterSnapshot(s.engine.Latest(), allowed, all)
	if !all {
		visibleGrows := make([]domain.GrowView, 0, len(snapshot.Grows))
		for _, grow := range snapshot.Grows {
			for _, environment := range grow.Environments {
				if allowed[environment.ID] {
					visibleGrows = append(visibleGrows, grow)
					break
				}
			}
		}
		snapshot.Grows = visibleGrows
	}
	activities, err := s.store.Activities("", "", nil, nil, 100, 0)
	if err != nil {
		return nil, err
	}
	if !all {
		filtered := make([]domain.Activity, 0, 25)
		for _, activity := range activities {
			if activity.EnvironmentID == "" || allowed[activity.EnvironmentID] {
				filtered = append(filtered, activity)
				if len(filtered) == 25 {
					break
				}
			}
		}
		activities = filtered
	} else if len(activities) > 25 {
		activities = activities[:25]
	}
	return json.Marshal(map[string]any{
		"scope": "all GrowRig", "generatedAt": time.Now(),
		"currentState": snapshot, "recentActivity": activities,
	})
}

func (s *Server) growAIContext(growID string) ([]byte, error) {
	grow, ok, err := s.store.Grow(growID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errGrowNotFound
	}
	units, err := s.store.PlantUnits(growID)
	if err != nil {
		return nil, err
	}
	environments, err := s.store.Environments()
	if err != nil {
		return nil, err
	}
	envNames := map[string]string{}
	for _, environment := range environments {
		envNames[environment.ID] = environment.Name
	}

	type plantContext struct {
		Label       string              `json:"label"`
		Cultivar    string              `json:"cultivar,omitempty"`
		Tracking    domain.TrackingMode `json:"tracking"`
		Quantity    int                 `json:"quantity"`
		Status      domain.PlantStatus  `json:"status"`
		Environment string              `json:"environment,omitempty"`
		AgeDays     int                 `json:"ageDays"`
	}
	plants := make([]plantContext, 0, len(units))
	environmentIDs := map[string]bool{}
	for _, unit := range units {
		currentEnvironment := ""
		placements, err := s.store.PlacementsForUnit(unit.ID)
		if err != nil {
			return nil, err
		}
		for _, placement := range placements {
			if placement.EndedAt == nil {
				currentEnvironment = envNames[placement.EnvironmentID]
				environmentIDs[placement.EnvironmentID] = true
				break
			}
		}
		plants = append(plants, plantContext{Label: unit.Label, Cultivar: unit.Cultivar, Tracking: unit.Tracking, Quantity: unit.Quantity, Status: unit.Status, Environment: currentEnvironment, AgeDays: domain.DaysSince(unit.CreatedAt, time.Now())})
	}

	type environmentContext struct {
		Name         string                  `json:"name"`
		Health       domain.ControllerHealth `json:"health"`
		TemperatureC *float64                `json:"temperatureC,omitempty"`
		Humidity     *float64                `json:"humidity,omitempty"`
		CO2          *float64                `json:"co2Ppm,omitempty"`
		VPD          *float64                `json:"vpdKpa,omitempty"`
	}
	type environmentHistory struct {
		Name     string           `json:"name"`
		Readings []domain.Reading `json:"readings"`
	}
	live := s.engine.Latest()
	liveEnvironments := []environmentContext{}
	for _, environment := range live.Environments {
		if !environmentIDs[environment.ID] {
			continue
		}
		item := environmentContext{Name: environment.Name, Health: environment.Health}
		if environment.HasTemp {
			value := environment.TempC
			item.TemperatureC = &value
		}
		if environment.HasHum {
			value := environment.Humidity
			item.Humidity = &value
		}
		if environment.HasCO2 {
			value := environment.CO2
			item.CO2 = &value
		}
		if environment.HasClimate {
			value := environment.VPD
			item.VPD = &value
		}
		liveEnvironments = append(liveEnvironments, item)
	}
	// Keep the prompt bounded while still giving the assistant enough chart
	// context to compare recent conditions. Forty-eight buckets across seven
	// days is one averaged sample every 3.5 hours per occupied environment.
	history := []environmentHistory{}
	for environmentID := range environmentIDs {
		readings, err := s.store.ReadingsSince(environmentID, time.Now().Add(-7*24*time.Hour), 48)
		if err != nil {
			return nil, err
		}
		if len(readings) > 0 {
			history = append(history, environmentHistory{Name: envNames[environmentID], Readings: readings})
		}
	}
	care, err := s.store.CareEvents(growID, 25, 0)
	if err != nil {
		return nil, err
	}
	activities, err := s.store.Activities("", growID, nil, nil, 25, 0)
	if err != nil {
		return nil, err
	}

	context := struct {
		GeneratedAt    time.Time            `json:"generatedAt"`
		Grow           domain.Grow          `json:"grow"`
		Plants         []plantContext       `json:"plants"`
		Environments   []environmentContext `json:"currentEnvironments"`
		HistoryWindow  string               `json:"historyWindow"`
		ClimateHistory []environmentHistory `json:"climateHistory"`
		RecentCare     []domain.CareEvent   `json:"recentCare"`
		RecentActivity []domain.Activity    `json:"recentActivity"`
	}{
		GeneratedAt:    time.Now(),
		Grow:           grow,
		Plants:         plants,
		Environments:   liveEnvironments,
		HistoryWindow:  "last 7 days, downsampled to at most 48 averaged readings per environment",
		ClimateHistory: history,
		RecentCare:     care,
		RecentActivity: activities,
	}
	return json.Marshal(context)
}

func integrationChatContent(result any) (string, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	var response struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Response string `json:"response"`
		Choices  []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", err
	}
	content := strings.TrimSpace(response.Message.Content)
	if content == "" {
		content = strings.TrimSpace(response.Response)
	}
	if content == "" && len(response.Choices) > 0 {
		content = strings.TrimSpace(response.Choices[0].Message.Content)
	}
	if content == "" {
		return "", fmt.Errorf("AI provider returned no message content")
	}
	return content, nil
}
