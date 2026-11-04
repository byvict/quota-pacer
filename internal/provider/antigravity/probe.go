package antigravity

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"quota-pacer/internal/host"
)

type httpDoer interface {
	HTTPDo(ctx context.Context, req host.HTTPRequest) (host.HTTPResponse, error)
}

// Prober 通过宿主 HTTPDo 执行 Antigravity quota summary fresh probe。
type Prober struct {
	host  httpDoer
	clock clock
}

// NewProber 创建使用宿主 HTTPDo 和注入时钟的 Antigravity fresh prober。
func NewProber(hostAPI httpDoer, clockSource clock) Prober {
	if clockSource == nil {
		clockSource = realClock{}
	}
	return Prober{host: hostAPI, clock: clockSource}
}

// Probe 请求 Antigravity quota 并返回目标模型组 quota 结果。
func (p Prober) Probe(ctx context.Context, request ProbeRequest) ProbeResult {
	observedAt := p.clock.Now().UTC()
	var (
		lastStatus int
		lastErr    string
	)
	for _, url := range probeURLs(request.ModelGroup) {
		response, err := p.host.HTTPDo(ctx, host.HTTPRequest{AuthIndex: request.AuthIndex, Method: http.MethodPost, URL: url, Headers: probeHeaders(request), Body: probeBody(request)})
		if err != nil {
			return failedProbe(request, observedAt, "host http do failed")
		}
		lastStatus = response.StatusCode
		if response.StatusCode != http.StatusOK {
			continue
		}
		result := ParseAvailableModels(response.Body, observedAt, request.ModelGroup)
		if result.Status == StatusReady {
			result.AuthIndex = request.AuthIndex
			return result
		}
		if result.Error != "" {
			lastErr = result.Error
		}
	}
	if lastErr != "" {
		return failedProbe(request, observedAt, lastErr)
	}
	if lastStatus != 0 {
		if lastStatus == http.StatusOK {
			return failedProbe(request, observedAt, "target model group quota unavailable")
		}
		return failedProbe(request, observedAt, fmt.Sprintf("retrieve quota summary status %d", lastStatus))
	}
	return failedProbe(request, observedAt, "probe failed")
}

func probeURLs(group ModelGroup) []string {
	urls := make([]string, 0, len(fetchAvailableModelsURLs)+len(retrieveUserQuotaSummaryURLs))
	if group == ModelGroupClaudeGPT {
		urls = append(urls, retrieveUserQuotaSummaryURLs...)
		urls = append(urls, fetchAvailableModelsURLs...)
	} else {
		urls = append(urls, fetchAvailableModelsURLs...)
		urls = append(urls, retrieveUserQuotaSummaryURLs...)
	}
	return urls
}

func probeBody(request ProbeRequest) []byte {
	projectID := strings.TrimSpace(request.ProjectID)
	if projectID == "" {
		return nil
	}
	body, err := json.Marshal(struct {
		Project string `json:"project"`
	}{Project: projectID})
	if err != nil {
		return nil
	}
	return body
}

func probeHeaders(request ProbeRequest) host.Header {
	headers := host.Header{"Accept": []string{"application/json"}, "Authorization": []string{"Bearer $TOKEN$"}, "Content-Type": []string{"application/json"}, "User-Agent": []string{"antigravity/cli/1.0.8 darwin/arm64"}}
	if token := strings.TrimSpace(request.AccessToken); token != "" {
		headers["Authorization"] = []string{"Bearer " + token}
	}
	return headers
}

func failedProbe(request ProbeRequest, observedAt time.Time, message string) ProbeResult {
	result := failedResult(observedAt, request.ModelGroup, safeError(message))
	result.AuthIndex = request.AuthIndex
	return result
}

func safeError(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "probe failed"
	}
	return trimmed
}
