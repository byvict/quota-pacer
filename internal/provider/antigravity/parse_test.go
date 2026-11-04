package antigravity

import (
	"testing"
	"time"

	"quota-pacer/internal/config"
	"quota-pacer/internal/core"
)

func TestParseAvailableModels_GeminiGroup(t *testing.T) {
	observedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)

	payload := `{
		"models": {
			"gemini-2.5-pro": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.85,
					"resetTime": "2026-08-20T15:00:00Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Provider != core.ProviderAntigravity {
		t.Errorf("expected ProviderAntigravity, got %v", result.Provider)
	}
	if result.Remaining == nil || *result.Remaining != 85 {
		t.Errorf("expected remaining 85, got %v", result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(resetAt) {
		t.Errorf("expected resetAt %v, got %v", resetAt, result.ResetAt)
	}
}

func TestParseAvailableModels_ClaudeGPTGroup(t *testing.T) {
	observedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC)

	payload := `{
		"models": {
			"claude-3-7-sonnet": {
				"modelProvider": "anthropic",
				"quotaInfo": {
					"remainingFraction": 0.60,
					"resetTime": "2026-08-20T16:00:00Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupClaudeGPT)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Remaining == nil || *result.Remaining != 60 {
		t.Errorf("expected remaining 60, got %v", result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(resetAt) {
		t.Errorf("expected resetAt %v, got %v", resetAt, result.ResetAt)
	}
}

func TestParseAvailableModels_GoogleQuotaGroups_GeminiWeeklyPriority(t *testing.T) {
	observedAt := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	weeklyReset := time.Date(2026, 8, 31, 6, 25, 13, 0, time.UTC)

	payload := `{
		"groups": [
			{
				"displayName": "Gemini Models",
				"description": "Models within this group: Gemini Flash, Gemini Pro",
				"buckets": [
					{
						"bucketId": "gemini-weekly",
						"displayName": "Weekly Limit Remaining",
						"window": "weekly",
						"resetTime": "2026-08-31T06:25:13Z",
						"remainingFraction": 0.8173522
					},
					{
						"bucketId": "gemini-5h",
						"displayName": "Five Hour Limit Remaining",
						"window": "5h",
						"resetTime": "2026-08-24T23:10:04Z",
						"remainingFraction": 0.9478041
					}
				]
			},
			{
				"displayName": "Claude and GPT models",
				"description": "Models within this group: Claude Opus, Claude Sonnet, GPT-OSS",
				"buckets": [
					{
						"bucketId": "3p-weekly",
						"displayName": "Weekly Limit Remaining",
						"window": "weekly",
						"resetTime": "2026-08-31T06:36:05Z",
						"remainingFraction": 1
					},
					{
						"bucketId": "3p-5h",
						"displayName": "Five Hour Limit Remaining",
						"window": "5h",
						"resetTime": "2026-08-24T23:19:10Z",
						"remainingFraction": 1
					}
				]
			}
		]
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Provider != core.ProviderAntigravity {
		t.Errorf("expected ProviderAntigravity, got %v", result.Provider)
	}
	if result.Remaining == nil || *result.Remaining != 82 {
		t.Errorf("expected remaining 82 (weekly quota), got %v", *result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(weeklyReset) {
		t.Errorf("expected resetAt %v, got %v", weeklyReset, result.ResetAt)
	}
	if result.LongWindowResetAt == nil || !result.LongWindowResetAt.Equal(weeklyReset) {
		t.Errorf("expected LongWindowResetAt %v, got %v", weeklyReset, result.LongWindowResetAt)
	}
	fiveHourReset := time.Date(2026, 8, 24, 23, 10, 4, 0, time.UTC)
	if result.ShortWindowRemaining == nil || *result.ShortWindowRemaining != 95 {
		t.Errorf("expected ShortWindowRemaining 95, got %v", result.ShortWindowRemaining)
	}
	if result.ShortWindowResetAt == nil || !result.ShortWindowResetAt.Equal(fiveHourReset) {
		t.Errorf("expected ShortWindowResetAt %v, got %v", fiveHourReset, result.ShortWindowResetAt)
	}
	if result.LongWindowRemaining == nil || *result.LongWindowRemaining != 82 {
		t.Errorf("expected LongWindowRemaining 82, got %v", result.LongWindowRemaining)
	}
}

func TestParseAvailableModels_GoogleQuotaGroups_FiveHourDepleted(t *testing.T) {
	observedAt := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	fiveHourReset := time.Date(2026, 8, 24, 23, 10, 4, 0, time.UTC)

	payload := `{
		"groups": [
			{
				"displayName": "Gemini Models",
				"description": "Models within this group: Gemini Flash, Gemini Pro",
				"buckets": [
					{
						"bucketId": "gemini-weekly",
						"displayName": "Weekly Limit Remaining",
						"window": "weekly",
						"resetTime": "2026-08-31T06:25:13Z",
						"remainingFraction": 0.8173522
					},
					{
						"bucketId": "gemini-5h",
						"displayName": "Five Hour Limit Remaining",
						"window": "5h",
						"resetTime": "2026-08-24T23:10:04Z",
						"remainingFraction": 0
					}
				]
			}
		]
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Remaining == nil || *result.Remaining != 0 {
		t.Errorf("expected remaining 0 when 5h is depleted, got %v", *result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(fiveHourReset) {
		t.Errorf("expected resetAt %v, got %v", fiveHourReset, result.ResetAt)
	}
	if result.ShortWindowRemaining == nil || *result.ShortWindowRemaining != 0 {
		t.Errorf("expected ShortWindowRemaining 0, got %v", result.ShortWindowRemaining)
	}
	if result.ShortWindowResetAt == nil || !result.ShortWindowResetAt.Equal(fiveHourReset) {
		t.Errorf("expected ShortWindowResetAt %v, got %v", fiveHourReset, result.ShortWindowResetAt)
	}
	if result.LongWindowRemaining == nil || *result.LongWindowRemaining != 82 {
		t.Errorf("expected LongWindowRemaining 82, got %v", result.LongWindowRemaining)
	}
}

func TestParseAvailableModels_SingleWindow_FiveHourInfer(t *testing.T) {
	observedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	resetAt := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)

	payload := `{
		"models": {
			"gemini-2.5-pro": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.85,
					"resetTime": "2026-08-20T15:00:00Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Window != WindowFiveHour {
		t.Errorf("expected WindowFiveHour, got %v", result.Window)
	}
	if result.ShortWindowRemaining == nil || *result.ShortWindowRemaining != 85 {
		t.Errorf("expected ShortWindowRemaining 85, got %v", result.ShortWindowRemaining)
	}
	if result.ShortWindowResetAt == nil || !result.ShortWindowResetAt.Equal(resetAt) {
		t.Errorf("expected ShortWindowResetAt %v, got %v", resetAt, result.ShortWindowResetAt)
	}
	if result.LongWindowRemaining != nil {
		t.Errorf("expected LongWindowRemaining nil, got %v", *result.LongWindowRemaining)
	}
	if result.LongWindowResetAt != nil {
		t.Errorf("expected LongWindowResetAt nil, got %v", *result.LongWindowResetAt)
	}
}

func TestParseAvailableModels_FetchAvailableModelsPayload(t *testing.T) {
	observedAt := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	weeklyReset := time.Date(2026, 8, 31, 6, 25, 13, 0, time.UTC)

	payload := `{
		"models": {
			"gemini-2.5-flash": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.16904838,
					"resetTime": "2026-08-31T06:25:13Z"
				}
			},
			"gemini-2.5-pro": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.16904838,
					"resetTime": "2026-08-31T06:25:13Z"
				}
			},
			"gemini-3.8-flash-tiered": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.16904838,
					"resetTime": "2026-08-31T06:25:13Z"
				}
			},
			"claude-3-7-sonnet": {
				"modelProvider": "anthropic",
				"quotaInfo": {
					"remainingFraction": 0.75,
					"resetTime": "2026-08-31T06:25:13Z"
				}
			}
		}
	}`

	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Provider != core.ProviderAntigravity {
		t.Errorf("expected ProviderAntigravity, got %v", result.Provider)
	}
	if result.Remaining == nil || *result.Remaining != 17 {
		t.Errorf("expected remaining 17 (rounded up from 0.16904838), got %v", result.Remaining)
	}
	if result.ResetAt == nil || !result.ResetAt.Equal(weeklyReset) {
		t.Errorf("expected resetAt %v, got %v", weeklyReset, result.ResetAt)
	}
	if result.LongWindowResetAt == nil || !result.LongWindowResetAt.Equal(weeklyReset) {
		t.Errorf("expected LongWindowResetAt %v, got %v", weeklyReset, result.LongWindowResetAt)
	}
	if result.LongWindowRemaining == nil || *result.LongWindowRemaining != 17 {
		t.Errorf("expected LongWindowRemaining 17, got %v", result.LongWindowRemaining)
	}
	if result.Window != WindowWeekly {
		t.Errorf("expected WindowWeekly, got %v", result.Window)
	}
	if result.ShortWindowRemaining != nil {
		t.Errorf("expected ShortWindowRemaining nil for weekly-only model, got %v", *result.ShortWindowRemaining)
	}
	if len(result.Windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(result.Windows))
	}
	if result.Windows[0].Name != "weekly" || result.Windows[0].Duration != 7*24*time.Hour {
		t.Errorf("expected weekly window of 7 days, got %+v", result.Windows[0])
	}
}

func TestParseAvailableModels_TargetModelGroupUnavailable(t *testing.T) {
	observedAt := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	payload := `{
		"models": {
			"claude-3-7-sonnet": {
				"modelProvider": "anthropic",
				"quotaInfo": {
					"remainingFraction": 0.75,
					"resetTime": "2026-08-31T06:25:13Z"
				}
			}
		}
	}`
	result := ParseAvailableModels([]byte(payload), observedAt, config.AntigravityModelGroupGemini)
	if result.Status != StatusProbeFailed {
		t.Errorf("expected StatusProbeFailed, got %v", result.Status)
	}
	if result.Error != "target model group quota unavailable" {
		t.Errorf("expected 'target model group quota unavailable', got %q", result.Error)
	}
}
