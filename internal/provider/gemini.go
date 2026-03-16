package provider

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

// Gemini implements the Provider interface for Google's Gemini API.
// Gemini has a unique API format where the model is in the URL path
// and the request/response structure differs from OpenAI.
type Gemini struct {
	upstream string
}

// NewGemini creates a Gemini provider.
func NewGemini(upstream string) *Gemini {
	if upstream == "" {
		upstream = "https://generativelanguage.googleapis.com"
	}
	return &Gemini{upstream: upstream}
}

func (g *Gemini) Name() string        { return "gemini" }
func (g *Gemini) UpstreamURL() string { return g.upstream }
func (g *Gemini) PathPrefix() string  { return "/gemini" }

// modelFromPathRe extracts the model name from Gemini URL paths like:
//
//	/gemini/v1beta/models/gemini-2.5-pro:generateContent
//	/gemini/v1beta/models/gemini-2.5-pro:streamGenerateContent
var modelFromPathRe = regexp.MustCompile(`/models/([^/:]+)`)

func (g *Gemini) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/gemini/")
}

// RewritePath strips the /gemini prefix so upstream sees the native path.
func (g *Gemini) RewritePath(path string) string {
	return strings.TrimPrefix(path, "/gemini")
}

// geminiRequest is the minimal subset of a Gemini generateContent request.
type geminiRequest struct {
	GenerationConfig *struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"generationConfig,omitempty"`
}

func (g *Gemini) ParseRequest(body []byte) (*RequestMeta, error) {
	var req geminiRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	meta := &RequestMeta{}
	if req.GenerationConfig != nil {
		meta.MaxTokens = req.GenerationConfig.MaxOutputTokens
	}
	// Model is extracted from the URL path, not the body — it's set by
	// the proxy after matching. We leave it empty here; the caller
	// extracts it from the path via ExtractGeminiModel.
	return meta, nil
}

// ExtractGeminiModel extracts the model name from a Gemini API URL path.
func ExtractGeminiModel(path string) string {
	m := modelFromPathRe.FindStringSubmatch(path)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// geminiResponse is the minimal subset of a Gemini generateContent response.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata,omitempty"`
	ModelVersion string `json:"modelVersion,omitempty"`
}

func (g *Gemini) ParseResponse(body []byte) (*ResponseMeta, error) {
	var resp geminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	meta := &ResponseMeta{
		Model: resp.ModelVersion,
	}
	if resp.UsageMetadata != nil {
		meta.InputTokens = resp.UsageMetadata.PromptTokenCount
		meta.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
		meta.TotalTokens = resp.UsageMetadata.TotalTokenCount
	}
	return meta, nil
}

// Gemini streams newline-delimited JSON arrays. Each chunk is a complete
// JSON object within the array. Usage appears in the final chunk.
func (g *Gemini) ParseStreamChunk(_ string, data []byte) (*StreamChunkMeta, error) {
	// Gemini stream chunks may be wrapped in array syntax; strip leading [ or ,
	data = stripJSONArrayWrapper(data)
	if len(data) == 0 {
		return &StreamChunkMeta{}, nil
	}

	var resp geminiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	meta := &StreamChunkMeta{
		Model: resp.ModelVersion,
	}

	// Extract text from candidates.
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		meta.Text = resp.Candidates[0].Content.Parts[0].Text
	}

	if resp.UsageMetadata != nil {
		meta.InputTokens = resp.UsageMetadata.PromptTokenCount
		meta.OutputTokens = resp.UsageMetadata.CandidatesTokenCount
		if meta.OutputTokens > 0 && meta.InputTokens > 0 {
			meta.Done = true
		}
	}

	return meta, nil
}

// stripJSONArrayWrapper removes leading/trailing array brackets and commas
// from Gemini's streamed JSON array format.
func stripJSONArrayWrapper(data []byte) []byte {
	s := strings.TrimSpace(string(data))
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimPrefix(s, ",")
	s = strings.TrimSuffix(s, ",")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return []byte(s)
}
