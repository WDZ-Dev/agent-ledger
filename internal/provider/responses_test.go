package provider

import "testing"

func TestParseResponsesAPIRequest(t *testing.T) {
	o := NewOpenAI("")
	body := []byte(`{"model":"gpt-5","input":"hello","max_output_tokens":500,"stream":true}`)
	meta, err := o.ParseRequest(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-5" {
		t.Errorf("model = %q, want gpt-5", meta.Model)
	}
	if meta.MaxTokens != 500 {
		t.Errorf("max_tokens = %d, want 500", meta.MaxTokens)
	}
	if !meta.Stream {
		t.Error("stream = false, want true")
	}
}

func TestParseResponsesAPIResponse(t *testing.T) {
	o := NewOpenAI("")
	body := []byte(`{"model":"gpt-5","usage":{"input_tokens":200,"output_tokens":100,"total_tokens":300}}`)
	meta, err := o.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.InputTokens != 200 {
		t.Errorf("input = %d, want 200", meta.InputTokens)
	}
	if meta.OutputTokens != 100 {
		t.Errorf("output = %d, want 100", meta.OutputTokens)
	}
}

func TestParseResponsesAPIStreamChunk(t *testing.T) {
	o := NewOpenAI("")

	// response.completed event
	data := []byte(`{"type":"response.completed","response":{"model":"gpt-5","usage":{"input_tokens":150,"output_tokens":75,"total_tokens":225}}}`)
	meta, err := o.ParseStreamChunk("", data)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-5" {
		t.Errorf("model = %q, want gpt-5", meta.Model)
	}
	if meta.InputTokens != 150 {
		t.Errorf("input = %d, want 150", meta.InputTokens)
	}
	if meta.OutputTokens != 75 {
		t.Errorf("output = %d, want 75", meta.OutputTokens)
	}
	if !meta.Done {
		t.Error("done = false, want true")
	}
}

func TestParseChatCompletionsStillWorks(t *testing.T) {
	o := NewOpenAI("")
	// Ensure traditional chat completions response still parses
	body := []byte(`{"model":"gpt-4o","usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`)
	meta, err := o.ParseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if meta.InputTokens != 100 {
		t.Errorf("input = %d, want 100", meta.InputTokens)
	}
	if meta.OutputTokens != 50 {
		t.Errorf("output = %d, want 50", meta.OutputTokens)
	}
}
