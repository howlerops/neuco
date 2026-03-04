package nango

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/neuco-ai/neuco/internal/domain"
	"github.com/neuco-ai/neuco/internal/store"
)

// SyncService fetches data from third-party integrations via the Nango proxy
// and converts the raw API responses into domain.Signal values that can be
// persisted by the caller.
type SyncService struct {
	nango *Client
	store *store.Store
}

// NewSyncService constructs a SyncService.
func NewSyncService(nango *Client, store *store.Store) *SyncService {
	return &SyncService{nango: nango, store: store}
}

// SyncGong fetches recent Gong call recordings via the Nango proxy and returns
// them as call-transcript signals. The Gong Calls API is used:
// GET /v2/calls — returns a page of calls.
func (s *SyncService) SyncGong(ctx context.Context, connectionID string, projectID uuid.UUID) ([]domain.Signal, error) {
	const providerConfigKey = "gong"

	resp, err := s.nango.Proxy(ctx, http.MethodGet, providerConfigKey, connectionID, "/v2/calls", nil)
	if err != nil {
		return nil, fmt.Errorf("nango.SyncGong: proxy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("nango.SyncGong: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Calls []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			StartedAt string `json:"started"`
			Duration  int    `json:"duration"`
			URL       string `json:"url"`
		} `json:"calls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("nango.SyncGong: decode response: %w", err)
	}

	signals := make([]domain.Signal, 0, len(result.Calls))
	for _, call := range result.Calls {
		occurredAt := time.Now().UTC()
		if call.StartedAt != "" {
			if t, err := time.Parse(time.RFC3339, call.StartedAt); err == nil {
				occurredAt = t
			}
		}

		content := call.Title
		if content == "" {
			content = fmt.Sprintf("Gong call %s", call.ID)
		}

		meta, _ := json.Marshal(map[string]any{
			"call_id":  call.ID,
			"duration": call.Duration,
			"url":      call.URL,
			"provider": "gong",
		})

		signals = append(signals, domain.Signal{
			ID:         uuid.New(),
			ProjectID:  projectID,
			Source:     domain.SignalSourceGong,
			SourceRef:  call.ID,
			Type:       domain.SignalTypeCallTranscript,
			Content:    content,
			Metadata:   json.RawMessage(meta),
			OccurredAt: occurredAt,
		})
	}

	slog.Info("nango.SyncGong: fetched calls",
		"connection_id", connectionID,
		"project_id", projectID,
		"count", len(signals),
	)

	return signals, nil
}

// SyncIntercom fetches recent Intercom conversations via the Nango proxy and
// returns them as support-ticket signals.
// GET /conversations — returns a list of conversations.
func (s *SyncService) SyncIntercom(ctx context.Context, connectionID string, projectID uuid.UUID) ([]domain.Signal, error) {
	const providerConfigKey = "intercom"

	resp, err := s.nango.Proxy(ctx, http.MethodGet, providerConfigKey, connectionID, "/conversations", nil)
	if err != nil {
		return nil, fmt.Errorf("nango.SyncIntercom: proxy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("nango.SyncIntercom: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Conversations []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			CreatedAt   int64  `json:"created_at"`
			Body        string `json:"body"`
			State       string `json:"state"`
			ContactType string `json:"type"`
		} `json:"conversations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("nango.SyncIntercom: decode response: %w", err)
	}

	signals := make([]domain.Signal, 0, len(result.Conversations))
	for _, conv := range result.Conversations {
		occurredAt := time.Now().UTC()
		if conv.CreatedAt > 0 {
			occurredAt = time.Unix(conv.CreatedAt, 0).UTC()
		}

		content := conv.Body
		if content == "" {
			content = conv.Title
		}
		if content == "" {
			content = fmt.Sprintf("Intercom conversation %s", conv.ID)
		}

		meta, _ := json.Marshal(map[string]any{
			"conversation_id": conv.ID,
			"title":           conv.Title,
			"state":           conv.State,
			"provider":        "intercom",
		})

		signals = append(signals, domain.Signal{
			ID:         uuid.New(),
			ProjectID:  projectID,
			Source:     domain.SignalSourceIntercom,
			SourceRef:  conv.ID,
			Type:       domain.SignalTypeSupportTicket,
			Content:    content,
			Metadata:   json.RawMessage(meta),
			OccurredAt: occurredAt,
		})
	}

	slog.Info("nango.SyncIntercom: fetched conversations",
		"connection_id", connectionID,
		"project_id", projectID,
		"count", len(signals),
	)

	return signals, nil
}

// SyncSlack fetches recent Slack messages from a configured channel via the
// Nango proxy and returns them as slack-message signals.
// GET /conversations.history — returns the message history of the first
// channel the token can access. In production the channel ID should be stored
// in the integration's config; here we use the general conversations list to
// discover a channel automatically.
func (s *SyncService) SyncSlack(ctx context.Context, connectionID string, projectID uuid.UUID) ([]domain.Signal, error) {
	const providerConfigKey = "slack"

	// Fetch recent messages from the conversations.history endpoint.
	// A real implementation would parameterise the channel; we use the first
	// channel returned by conversations.list as a sensible default.
	channelID, err := s.resolveSlackChannel(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("nango.SyncSlack: resolve channel: %w", err)
	}

	path := fmt.Sprintf("/conversations.history?channel=%s&limit=50", channelID)
	resp, err := s.nango.Proxy(ctx, http.MethodGet, providerConfigKey, connectionID, path, nil)
	if err != nil {
		return nil, fmt.Errorf("nango.SyncSlack: proxy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("nango.SyncSlack: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK       bool `json:"ok"`
		Messages []struct {
			Ts   string `json:"ts"`
			Text string `json:"text"`
			User string `json:"user"`
			Type string `json:"type"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("nango.SyncSlack: decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("nango.SyncSlack: Slack API returned ok=false for channel %s", channelID)
	}

	signals := make([]domain.Signal, 0, len(result.Messages))
	for _, msg := range result.Messages {
		if msg.Type != "message" || msg.Text == "" {
			continue
		}

		occurredAt := time.Now().UTC()
		if msg.Ts != "" {
			// Slack timestamps are Unix seconds with fractional part: "1234567890.123456"
			var unixSec float64
			if _, err := fmt.Sscanf(msg.Ts, "%f", &unixSec); err == nil {
				occurredAt = time.Unix(int64(unixSec), 0).UTC()
			}
		}

		meta, _ := json.Marshal(map[string]any{
			"ts":         msg.Ts,
			"user":       msg.User,
			"channel_id": channelID,
			"provider":   "slack",
		})

		signals = append(signals, domain.Signal{
			ID:         uuid.New(),
			ProjectID:  projectID,
			Source:     domain.SignalSourceSlack,
			SourceRef:  msg.Ts,
			Type:       domain.SignalTypeSlackMessage,
			Content:    msg.Text,
			Metadata:   json.RawMessage(meta),
			OccurredAt: occurredAt,
		})
	}

	slog.Info("nango.SyncSlack: fetched messages",
		"connection_id", connectionID,
		"project_id", projectID,
		"channel_id", channelID,
		"count", len(signals),
	)

	return signals, nil
}

// resolveSlackChannel returns the ID of the first public channel the token
// has access to. This is used as a fallback when no channel is configured
// explicitly on the integration.
func (s *SyncService) resolveSlackChannel(ctx context.Context, connectionID string) (string, error) {
	const providerConfigKey = "slack"

	resp, err := s.nango.Proxy(ctx, http.MethodGet, providerConfigKey, connectionID, "/conversations.list?limit=1&exclude_archived=true", nil)
	if err != nil {
		return "", fmt.Errorf("proxy request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK       bool `json:"ok"`
		Channels []struct {
			ID string `json:"id"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if !result.OK || len(result.Channels) == 0 {
		return "", fmt.Errorf("no accessible Slack channels found")
	}

	return result.Channels[0].ID, nil
}

// SyncGeneric is a best-effort sync for providers that do not have a dedicated
// sync method. It calls GET / on the provider's API through the Nango proxy,
// reads the raw JSON body, and stores it as a single signal of type "event".
// This is useful for new integrations before a dedicated parser is written.
func (s *SyncService) SyncGeneric(
	ctx context.Context,
	providerConfigKey string,
	connectionID string,
	projectID uuid.UUID,
) ([]domain.Signal, error) {
	resp, err := s.nango.Proxy(ctx, http.MethodGet, providerConfigKey, connectionID, "/", nil)
	if err != nil {
		return nil, fmt.Errorf("nango.SyncGeneric: proxy request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("nango.SyncGeneric: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nango.SyncGeneric: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	content := string(body)
	if content == "" {
		content = fmt.Sprintf("sync from %s", providerConfigKey)
	}

	meta, _ := json.Marshal(map[string]any{
		"provider":     providerConfigKey,
		"connection_id": connectionID,
		"status":       resp.StatusCode,
	})

	sig := domain.Signal{
		ID:         uuid.New(),
		ProjectID:  projectID,
		Source:     domain.SignalSource(providerConfigKey),
		SourceRef:  connectionID,
		Type:       domain.SignalTypeEvent,
		Content:    content,
		Metadata:   json.RawMessage(meta),
		OccurredAt: time.Now().UTC(),
	}

	slog.Info("nango.SyncGeneric: fetched data",
		"provider", providerConfigKey,
		"connection_id", connectionID,
		"project_id", projectID,
	)

	return []domain.Signal{sig}, nil
}
