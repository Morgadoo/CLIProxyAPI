// Package registry provides model definitions and lookup helpers for various AI providers.
// Static model metadata is loaded from the embedded models.json file and can be refreshed from network.
package registry

import (
	"strings"
)

// staticModelsJSON mirrors the top-level structure of models.json.
type staticModelsJSON struct {
	Claude      []*ModelInfo `json:"claude"`
	Gemini      []*ModelInfo `json:"gemini"`
	Vertex      []*ModelInfo `json:"vertex"`
	GeminiCLI   []*ModelInfo `json:"gemini-cli"`
	AIStudio    []*ModelInfo `json:"aistudio"`
	CodexFree   []*ModelInfo `json:"codex-free"`
	CodexTeam   []*ModelInfo `json:"codex-team"`
	CodexPlus   []*ModelInfo `json:"codex-plus"`
	CodexPro    []*ModelInfo `json:"codex-pro"`
	Kimi        []*ModelInfo `json:"kimi"`
	Antigravity []*ModelInfo `json:"antigravity"`
	RedPill     []*ModelInfo `json:"redpill"`
}

// GetClaudeModels returns the standard Claude model definitions.
func GetClaudeModels() []*ModelInfo {
	return cloneModelInfos(getModels().Claude)
}

// GetGeminiModels returns the standard Gemini model definitions.
func GetGeminiModels() []*ModelInfo {
	return cloneModelInfos(getModels().Gemini)
}

// GetGeminiVertexModels returns Gemini model definitions for Vertex AI.
func GetGeminiVertexModels() []*ModelInfo {
	return cloneModelInfos(getModels().Vertex)
}

// GetGeminiCLIModels returns Gemini model definitions for the Gemini CLI.
func GetGeminiCLIModels() []*ModelInfo {
	return cloneModelInfos(getModels().GeminiCLI)
}

// GetAIStudioModels returns model definitions for AI Studio.
func GetAIStudioModels() []*ModelInfo {
	return cloneModelInfos(getModels().AIStudio)
}

// GetCodexFreeModels returns model definitions for the Codex free plan tier.
func GetCodexFreeModels() []*ModelInfo {
	return cloneModelInfos(getModels().CodexFree)
}

// GetCodexTeamModels returns model definitions for the Codex team plan tier.
func GetCodexTeamModels() []*ModelInfo {
	return cloneModelInfos(getModels().CodexTeam)
}

// GetCodexPlusModels returns model definitions for the Codex plus plan tier.
func GetCodexPlusModels() []*ModelInfo {
	return cloneModelInfos(getModels().CodexPlus)
}

// GetCodexProModels returns model definitions for the Codex pro plan tier.
func GetCodexProModels() []*ModelInfo {
	return cloneModelInfos(getModels().CodexPro)
}

// GetKimiModels returns the standard Kimi (Moonshot AI) model definitions.
func GetKimiModels() []*ModelInfo {
	return cloneModelInfos(getModels().Kimi)
}

// GetRedPillModels returns the standard RedPill model definitions.
func GetRedPillModels() []*ModelInfo {
	models := getModels().RedPill
	if len(models) > 0 {
		return cloneModelInfos(models)
	}
	// Fallback: return hardcoded models if not in models.json
	return getRedPillDefaultModels()
}

// getRedPillDefaultModels returns hardcoded RedPill model definitions.
func getRedPillDefaultModels() []*ModelInfo {
	return []*ModelInfo{
		{ID: "deepseek/deepseek-r1-0528", DisplayName: "DeepSeek R1 0528", ContextLength: 163840},
		{ID: "deepseek/deepseek-v3.2", DisplayName: "DeepSeek V3.2", ContextLength: 163840},
		{ID: "deepseek/deepseek-chat-v3.1", DisplayName: "DeepSeek V3.1", ContextLength: 163840},
		{ID: "qwen/qwen3-coder-480b-a35b-instruct", DisplayName: "Qwen3 Coder 480B", ContextLength: 262000},
		{ID: "qwen/qwen3.5-397b-a17b", DisplayName: "Qwen3.5 397B A17B", ContextLength: 262144},
		{ID: "qwen/qwen3.5-27b", DisplayName: "Qwen3.5 27B", ContextLength: 262144},
		{ID: "qwen/qwen3-30b-a3b-instruct-2507", DisplayName: "Qwen3 30B A3B", ContextLength: 262144},
		{ID: "qwen/qwen-2.5-7b-instruct", DisplayName: "Qwen2.5 7B", ContextLength: 32768},
		{ID: "qwen/qwen3-vl-30b-a3b-instruct", DisplayName: "Qwen3 VL 30B", ContextLength: 128000},
		{ID: "openai/gpt-oss-120b", DisplayName: "GPT OSS 120B", ContextLength: 131072},
		{ID: "google/gemma-3-27b-it", DisplayName: "Gemma 3 27B", ContextLength: 53920},
		{ID: "meta-llama/llama-3.3-70b-instruct", DisplayName: "Llama 3.3 70B", ContextLength: 131072},
		{ID: "moonshotai/kimi-k2-thinking", DisplayName: "Kimi K2 Thinking", ContextLength: 262144},
		{ID: "moonshotai/kimi-k2.5", DisplayName: "Kimi K2.5", ContextLength: 262144},
		{ID: "z-ai/glm-5", DisplayName: "GLM 5", ContextLength: 202752},
		{ID: "z-ai/glm-4.7", DisplayName: "GLM 4.7", ContextLength: 131072},
		{ID: "z-ai/glm-4.7-flash", DisplayName: "GLM 4.7 Flash", ContextLength: 202752},
		{ID: "phala/uncensored-24b", DisplayName: "Venice Uncensored 24B", ContextLength: 32768},
	}
}

// GetAntigravityModels returns the standard Antigravity model definitions.
func GetAntigravityModels() []*ModelInfo {
	return cloneModelInfos(getModels().Antigravity)
}

// cloneModelInfos returns a shallow copy of the slice with each element deep-cloned.
func cloneModelInfos(models []*ModelInfo) []*ModelInfo {
	if len(models) == 0 {
		return nil
	}
	out := make([]*ModelInfo, len(models))
	for i, m := range models {
		out[i] = cloneModelInfo(m)
	}
	return out
}

// GetStaticModelDefinitionsByChannel returns static model definitions for a given channel/provider.
// It returns nil when the channel is unknown.
//
// Supported channels:
//   - claude
//   - gemini
//   - vertex
//   - gemini-cli
//   - aistudio
//   - codex
//   - kimi
//   - antigravity
func GetStaticModelDefinitionsByChannel(channel string) []*ModelInfo {
	key := strings.ToLower(strings.TrimSpace(channel))
	switch key {
	case "claude":
		return GetClaudeModels()
	case "gemini":
		return GetGeminiModels()
	case "vertex":
		return GetGeminiVertexModels()
	case "gemini-cli":
		return GetGeminiCLIModels()
	case "aistudio":
		return GetAIStudioModels()
	case "codex":
		return GetCodexProModels()
	case "kimi":
		return GetKimiModels()
	case "antigravity":
		return GetAntigravityModels()
	case "redpill":
		return GetRedPillModels()
	default:
		return nil
	}
}

// LookupStaticModelInfo searches all static model definitions for a model by ID.
// Returns nil if no matching model is found.
func LookupStaticModelInfo(modelID string) *ModelInfo {
	if modelID == "" {
		return nil
	}

	data := getModels()
	allModels := [][]*ModelInfo{
		data.Claude,
		data.Gemini,
		data.Vertex,
		data.GeminiCLI,
		data.AIStudio,
		data.CodexPro,
		data.Kimi,
		data.Antigravity,
	}
	for _, models := range allModels {
		for _, m := range models {
			if m != nil && m.ID == modelID {
				return cloneModelInfo(m)
			}
		}
	}

	return nil
}
