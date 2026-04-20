package management

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const (
	testModelsDefaultPrompt      = "Reply with exactly: OK"
	testModelsDefaultTimeout     = 20 * time.Second
	testModelsMaxTimeout         = 120 * time.Second
	testModelsDefaultConcurrency = 5
	testModelsMaxConcurrency     = 20
)

type testModelsRequest struct {
	Models      []string `json:"models"`
	Prompt      string   `json:"prompt"`
	Concurrency int      `json:"concurrency"`
	Timeout     int      `json:"timeout_seconds"`
	MaxTokens   int      `json:"max_tokens"`
}

type testModelResult struct {
	Model      string `json:"model"`
	Success    bool   `json:"success"`
	LatencyMS  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Reply      string `json:"reply,omitempty"`
	Error      string `json:"error,omitempty"`
	PromptTok  int    `json:"prompt_tokens,omitempty"`
	OutputTok  int    `json:"completion_tokens,omitempty"`
}

type testModelsResponse struct {
	Total      int               `json:"total"`
	Passed     int               `json:"passed"`
	Failed     int               `json:"failed"`
	DurationMS int64             `json:"duration_ms"`
	Results    []testModelResult `json:"results"`
}

// TestModels pings each configured model with a short chat completion and
// returns per-model success/latency/error, so operators can verify provider
// health from a single call.
func (h *Handler) TestModels(c *gin.Context) {
	var req testModelsRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
			return
		}
	}

	cfg := h.cfg
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server config unavailable"})
		return
	}

	apiKey := firstAPIKey(cfg.APIKeys)
	if apiKey == "" {
		c.JSON(http.StatusPreconditionFailed, gin.H{
			"error": "no client api-keys configured; add at least one entry under api-keys to enable model testing",
		})
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = testModelsDefaultPrompt
	}
	perModelTimeout := testModelsDefaultTimeout
	if req.Timeout > 0 {
		perModelTimeout = time.Duration(req.Timeout) * time.Second
		if perModelTimeout > testModelsMaxTimeout {
			perModelTimeout = testModelsMaxTimeout
		}
	}
	concurrency := testModelsDefaultConcurrency
	if req.Concurrency > 0 {
		concurrency = req.Concurrency
		if concurrency > testModelsMaxConcurrency {
			concurrency = testModelsMaxConcurrency
		}
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 20
	}

	port := cfg.Port
	if port <= 0 {
		port = 8317
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	models := req.Models
	if len(models) == 0 {
		listed, err := listProxyModels(c.Request.Context(), baseURL, apiKey)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to list models: " + err.Error()})
			return
		}
		models = listed
	}
	models = dedupeStrings(models)
	if len(models) == 0 {
		c.JSON(http.StatusOK, testModelsResponse{Results: []testModelResult{}})
		return
	}

	sem := make(chan struct{}, concurrency)
	results := make([]testModelResult, len(models))
	var wg sync.WaitGroup
	start := time.Now()

	for i, m := range models {
		i, m := i, m
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(c.Request.Context(), perModelTimeout)
			defer cancel()
			results[i] = testOneModel(ctx, baseURL, apiKey, m, prompt, maxTokens)
		}()
	}
	wg.Wait()

	passed := 0
	for _, r := range results {
		if r.Success {
			passed++
		}
	}

	c.JSON(http.StatusOK, testModelsResponse{
		Total:      len(results),
		Passed:     passed,
		Failed:     len(results) - passed,
		DurationMS: time.Since(start).Milliseconds(),
		Results:    results,
	})
}

func firstAPIKey(keys []string) string {
	for _, k := range keys {
		if s := strings.TrimSpace(k); s != "" {
			return s
		}
	}
	return ""
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func listProxyModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var ids []string
	gjson.GetBytes(body, "data.#.id").ForEach(func(_, v gjson.Result) bool {
		ids = append(ids, v.String())
		return true
	})
	return ids, nil
}

func testOneModel(ctx context.Context, baseURL, apiKey, model, prompt string, maxTokens int) testModelResult {
	start := time.Now()
	res := testModelResult{Model: model}

	payload := map[string]any{
		"model":      model,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens": maxTokens,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		res.Error = err.Error()
		res.LatencyMS = time.Since(start).Milliseconds()
		return res
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = err.Error()
		res.LatencyMS = time.Since(start).Milliseconds()
		return res
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	res.StatusCode = resp.StatusCode
	res.LatencyMS = time.Since(start).Milliseconds()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if msg := gjson.GetBytes(raw, "error.message").String(); msg != "" {
			res.Error = msg
		} else if msg := gjson.GetBytes(raw, "message").String(); msg != "" {
			res.Error = msg
		} else if msg := gjson.GetBytes(raw, "detail").String(); msg != "" {
			res.Error = msg
		} else {
			res.Error = truncate(string(raw), 240)
		}
		return res
	}

	content := gjson.GetBytes(raw, "choices.0.message.content").String()
	if content == "" {
		content = gjson.GetBytes(raw, "choices.0.text").String()
	}
	res.Reply = truncate(content, 200)
	res.PromptTok = int(gjson.GetBytes(raw, "usage.prompt_tokens").Int())
	res.OutputTok = int(gjson.GetBytes(raw, "usage.completion_tokens").Int())
	res.Success = true
	return res
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
