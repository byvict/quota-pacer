package antigravity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"quota-pacer/internal/host"
)

type mockHTTPDoer struct {
	mu       sync.Mutex
	requests []host.HTTPRequest
	fn       func(req host.HTTPRequest) (host.HTTPResponse, error)
}

func (m *mockHTTPDoer) HTTPDo(_ context.Context, req host.HTTPRequest) (host.HTTPResponse, error) {
	m.mu.Lock()
	m.requests = append(m.requests, req)
	fn := m.fn
	m.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return host.HTTPResponse{StatusCode: http.StatusOK}, nil
}

func (m *mockHTTPDoer) getRequests() []host.HTTPRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]host.HTTPRequest, len(m.requests))
	copy(cp, m.requests)
	return cp
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time {
	return f.now
}

func TestProbe_GeminiRoutesToFetchAvailableModels(t *testing.T) {
	fixedTime := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	geminiPayload := `{
		"models": {
			"gemini-2.5-pro": {
				"modelProvider": "google",
				"quotaInfo": {
					"remainingFraction": 0.85,
					"resetTime": "2026-08-31T10:00:00Z"
				}
			}
		}
	}`

	mock := &mockHTTPDoer{
		fn: func(req host.HTTPRequest) (host.HTTPResponse, error) {
			return host.HTTPResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(geminiPayload),
			}, nil
		},
	}

	prober := NewProber(mock, fixedClock{now: fixedTime})
	req := ProbeRequest{
		AuthIndex:   "auth-gemini-1",
		AccessToken: "secret-token-xyz",
		ProjectID:   "my-gemini-project",
		ModelGroup:  ModelGroupGemini,
	}

	result := prober.Probe(context.Background(), req)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.AuthIndex != "auth-gemini-1" {
		t.Errorf("expected AuthIndex 'auth-gemini-1', got %s", result.AuthIndex)
	}
	if result.Remaining == nil || *result.Remaining != 85 {
		t.Errorf("expected remaining 85, got %v", result.Remaining)
	}
	if result.Window != WindowWeekly {
		t.Errorf("expected WindowWeekly, got %v", result.Window)
	}

	requests := mock.getRequests()
	if len(requests) != 1 {
		t.Fatalf("expected exactly 1 HTTP request, got %d", len(requests))
	}

	firstReq := requests[0]
	if firstReq.URL != FetchAvailableModelsURL {
		t.Errorf("expected URL %s, got %s", FetchAvailableModelsURL, firstReq.URL)
	}
	if len(firstReq.Headers["Authorization"]) == 0 || firstReq.Headers["Authorization"][0] != "Bearer secret-token-xyz" {
		t.Errorf("expected Bearer token in headers, got %v", firstReq.Headers["Authorization"])
	}
	if len(firstReq.Headers["Content-Type"]) == 0 || firstReq.Headers["Content-Type"][0] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %v", firstReq.Headers["Content-Type"])
	}
	if !strings.Contains(string(firstReq.Body), `"project":"my-gemini-project"`) {
		t.Errorf("expected project in body, got %s", string(firstReq.Body))
	}
}

func TestProbe_ClaudeGPTRoutesToRetrieveUserQuotaSummary(t *testing.T) {
	fixedTime := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	claudePayload := `{
		"groups": [
			{
				"displayName": "Claude and GPT models",
				"buckets": [
					{
						"bucketId": "3p-5h",
						"window": "5h",
						"remainingFraction": 0.60,
						"resetTime": "2026-08-24T15:00:00Z"
					}
				]
			}
		]
	}`

	mock := &mockHTTPDoer{
		fn: func(req host.HTTPRequest) (host.HTTPResponse, error) {
			return host.HTTPResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(claudePayload),
			}, nil
		},
	}

	prober := NewProber(mock, fixedClock{now: fixedTime})
	req := ProbeRequest{
		AuthIndex:   "auth-claude-1",
		AccessToken: "secret-token-abc",
		ProjectID:   "my-claude-project",
		ModelGroup:  ModelGroupClaudeGPT,
	}

	result := prober.Probe(context.Background(), req)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Remaining == nil || *result.Remaining != 60 {
		t.Errorf("expected remaining 60, got %v", result.Remaining)
	}

	requests := mock.getRequests()
	if len(requests) != 1 {
		t.Fatalf("expected exactly 1 HTTP request, got %d", len(requests))
	}

	firstReq := requests[0]
	if firstReq.URL != RetrieveUserQuotaSummaryURL {
		t.Errorf("expected URL %s, got %s", RetrieveUserQuotaSummaryURL, firstReq.URL)
	}
	if len(firstReq.Headers["Authorization"]) == 0 || firstReq.Headers["Authorization"][0] != "Bearer secret-token-abc" {
		t.Errorf("expected Bearer token in headers, got %v", firstReq.Headers["Authorization"])
	}
	if !strings.Contains(string(firstReq.Body), `"project":"my-claude-project"`) {
		t.Errorf("expected project in body, got %s", string(firstReq.Body))
	}
}

func TestProbe_GeminiFallbackToRetrieveUserQuotaSummary(t *testing.T) {
	fixedTime := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	fallbackPayload := `{
		"groups": [
			{
				"displayName": "Gemini Models",
				"buckets": [
					{
						"bucketId": "gemini-weekly",
						"window": "weekly",
						"remainingFraction": 0.90,
						"resetTime": "2026-08-31T10:00:00Z"
					}
				]
			}
		]
	}`

	mock := &mockHTTPDoer{
		fn: func(req host.HTTPRequest) (host.HTTPResponse, error) {
			if strings.Contains(req.URL, "fetchAvailableModels") {
				return host.HTTPResponse{StatusCode: http.StatusNotFound}, nil
			}
			return host.HTTPResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(fallbackPayload),
			}, nil
		},
	}

	prober := NewProber(mock, fixedClock{now: fixedTime})
	req := ProbeRequest{
		AuthIndex:   "auth-fallback-1",
		AccessToken: "test-token",
		ModelGroup:  ModelGroupGemini,
	}

	result := prober.Probe(context.Background(), req)
	if result.Status != StatusReady {
		t.Fatalf("expected StatusReady via fallback, got %v (err: %s)", result.Status, result.Error)
	}
	if result.Remaining == nil || *result.Remaining != 90 {
		t.Errorf("expected remaining 90, got %v", result.Remaining)
	}

	requests := mock.getRequests()
	if len(requests) <= len(fetchAvailableModelsURLs) {
		t.Errorf("expected to try fetchAvailableModelsURLs then fallback, got %d requests", len(requests))
	}
}

func TestProbe_Failure_StatusCodes(t *testing.T) {
	statusCodes := []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError}

	for _, statusCode := range statusCodes {
		mock := &mockHTTPDoer{
			fn: func(_ host.HTTPRequest) (host.HTTPResponse, error) {
				return host.HTTPResponse{StatusCode: statusCode}, nil
			},
		}

		prober := NewProber(mock, nil)
		req := ProbeRequest{
			AuthIndex:   "auth-fail",
			AccessToken: "sensitive-secret-token",
			ModelGroup:  ModelGroupGemini,
		}

		result := prober.Probe(context.Background(), req)
		if result.Status != StatusProbeFailed {
			t.Errorf("status %d: expected StatusProbeFailed, got %v", statusCode, result.Status)
		}
		if strings.Contains(result.Error, "sensitive-secret-token") {
			t.Errorf("status %d: error leaked sensitive token: %s", statusCode, result.Error)
		}
		if !strings.Contains(result.Error, "retrieve quota summary status") {
			t.Errorf("status %d: expected status in error, got %s", statusCode, result.Error)
		}
	}
}

func TestProbe_Failure_NetworkTimeout(t *testing.T) {
	mock := &mockHTTPDoer{
		fn: func(_ host.HTTPRequest) (host.HTTPResponse, error) {
			return host.HTTPResponse{}, errors.New("context deadline exceeded")
		},
	}

	prober := NewProber(mock, nil)
	req := ProbeRequest{
		AuthIndex:   "auth-timeout",
		AccessToken: "sensitive-secret-token",
		ModelGroup:  ModelGroupGemini,
	}

	result := prober.Probe(context.Background(), req)
	if result.Status != StatusProbeFailed {
		t.Errorf("expected StatusProbeFailed, got %v", result.Status)
	}
	if result.Error != "host http do failed" {
		t.Errorf("expected 'host http do failed', got %s", result.Error)
	}
	if strings.Contains(result.Error, "sensitive-secret-token") {
		t.Errorf("error leaked sensitive token: %s", result.Error)
	}
}

func TestProbe_Failure_Status200_NoTargetModelGroup(t *testing.T) {
	claudeOnlyPayload := `{
		"models": {
			"claude-3-7-sonnet": {
				"modelProvider": "anthropic",
				"quotaInfo": {
					"remainingFraction": 0.85,
					"resetTime": "2026-08-31T10:00:00Z"
				}
			}
		}
	}`

	mock := &mockHTTPDoer{
		fn: func(_ host.HTTPRequest) (host.HTTPResponse, error) {
			return host.HTTPResponse{
				StatusCode: http.StatusOK,
				Body:       []byte(claudeOnlyPayload),
			}, nil
		},
	}

	prober := NewProber(mock, nil)
	req := ProbeRequest{
		AuthIndex:   "auth-no-gemini",
		AccessToken: "sensitive-token",
		ModelGroup:  ModelGroupGemini,
	}

	result := prober.Probe(context.Background(), req)
	if result.Status != StatusProbeFailed {
		t.Errorf("expected StatusProbeFailed, got %v", result.Status)
	}
	if result.Error != "target model group quota unavailable" {
		t.Errorf("expected 'target model group quota unavailable', got %q", result.Error)
	}
	if strings.Contains(result.Error, "200") {
		t.Errorf("error must not contain ambiguous status 200, got %q", result.Error)
	}
}
