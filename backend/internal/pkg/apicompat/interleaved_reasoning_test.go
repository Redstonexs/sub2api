package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicToResponsesPreservesSignedInterleavedThinking(t *testing.T) {
	var req AnthropicRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"model":"gpt-5.6",
		"max_tokens":2048,
		"messages":[
			{"role":"user","content":"begin"},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"first plan","signature":"enc-first"},
				{"type":"tool_use","id":"call_1","name":"lookup","input":{"q":"one"}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"call_1","content":"one"}
			]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"","signature":"enc-second"},
				{"type":"text","text":"checking again"},
				{"type":"tool_use","id":"call_2","name":"lookup","input":{"q":"two"}}
			]},
			{"role":"user","content":[
				{"type":"tool_result","tool_use_id":"call_2","content":"two"}
			]}
		]
	}`), &req))

	resp, err := AnthropicToResponses(&req)
	require.NoError(t, err)

	var items []map[string]any
	require.NoError(t, json.Unmarshal(resp.Input, &items))
	require.Len(t, items, 8)
	assert.Equal(t, []string{
		"message",
		"reasoning",
		"function_call",
		"function_call_output",
		"reasoning",
		"message",
		"function_call",
		"function_call_output",
	}, responseInputItemTypes(items))

	assert.Equal(t, "enc-first", items[1]["encrypted_content"])
	assert.Equal(t, []any{map[string]any{
		"type": "summary_text",
		"text": "first plan",
	}}, items[1]["summary"])
	assert.Equal(t, "enc-second", items[4]["encrypted_content"])
	assert.Equal(t, []any{}, items[4]["summary"])
	assert.Equal(t, "call_2", items[6]["call_id"])

	body, err := json.Marshal(resp)
	require.NoError(t, err)
	var requestJSON map[string]any
	require.NoError(t, json.Unmarshal(body, &requestJSON))
	reasoning, ok := requestJSON["reasoning"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "all_turns", reasoning["context"])
}

func responseInputItemTypes(items []map[string]any) []string {
	types := make([]string, 0, len(items))
	for _, item := range items {
		itemType, _ := item["type"].(string)
		types = append(types, itemType)
	}
	return types
}

func TestAnthropicToResponsesPreservesRedactedThinkingData(t *testing.T) {
	var req AnthropicRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"model":"gpt-5.6",
		"messages":[{"role":"assistant","content":[
			{"type":"redacted_thinking","data":"enc-redacted"},
			{"type":"tool_use","id":"call_1","name":"lookup","input":{}}
		]}]
	}`), &req))

	resp, err := AnthropicToResponses(&req)
	require.NoError(t, err)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(resp.Input, &items))
	require.Len(t, items, 2)
	assert.Equal(t, "reasoning", items[0]["type"])
	assert.Equal(t, "enc-redacted", items[0]["encrypted_content"])
	assert.Equal(t, []any{}, items[0]["summary"])
	assert.Equal(t, "function_call", items[1]["type"])
}

func TestResponsesToAnthropicPreservesInterleavedReasoningSignatures(t *testing.T) {
	resp := &ResponsesResponse{
		ID:     "resp_interleaved",
		Model:  "gpt-5.6",
		Status: "completed",
		Output: []ResponsesOutput{
			{
				Type:             "reasoning",
				EncryptedContent: "enc-first",
				Summary: []ResponsesSummary{
					{Type: "summary_text", Text: "first plan"},
				},
			},
			{Type: "function_call", CallID: "call_1", Name: "lookup", Arguments: `{"q":"one"}`},
			{Type: "reasoning", EncryptedContent: "enc-second", Summary: []ResponsesSummary{}},
			{
				Type: "message",
				Content: []ResponsesContentPart{
					{Type: "output_text", Text: "done"},
				},
			},
		},
	}

	anthropic := ResponsesToAnthropic(resp, "claude-opus-4-6")
	require.Len(t, anthropic.Content, 4)
	assert.Equal(t, []string{"thinking", "tool_use", "thinking", "text"}, []string{
		anthropic.Content[0].Type,
		anthropic.Content[1].Type,
		anthropic.Content[2].Type,
		anthropic.Content[3].Type,
	})

	body, err := json.Marshal(anthropic)
	require.NoError(t, err)
	var responseJSON map[string]any
	require.NoError(t, json.Unmarshal(body, &responseJSON))
	content, ok := responseJSON["content"].([]any)
	require.True(t, ok)
	first, ok := content[0].(map[string]any)
	require.True(t, ok)
	second, ok := content[2].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "enc-first", first["signature"])
	assert.Equal(t, "enc-second", second["signature"])
	assert.Equal(t, "", second["thinking"])
}

func TestResponsesStreamEmitsSignatureBeforeEachThinkingBlockStops(t *testing.T) {
	state := NewResponsesEventToAnthropicState()
	events := []ResponsesStreamEvent{
		{Type: "response.created", Response: &ResponsesResponse{ID: "resp_stream", Model: "gpt-5.6"}},
		{
			Type:        "response.output_item.added",
			OutputIndex: 0,
			Item:        &ResponsesOutput{Type: "reasoning", ID: "rs_1", Status: "in_progress"},
		},
		{Type: "response.reasoning_summary_text.delta", OutputIndex: 0, Delta: "first plan"},
		{Type: "response.reasoning_summary_text.done", OutputIndex: 0},
		{
			Type:        "response.output_item.done",
			OutputIndex: 0,
			Item: &ResponsesOutput{
				Type:             "reasoning",
				ID:               "rs_1",
				Status:           "completed",
				EncryptedContent: "enc-first",
			},
		},
		{
			Type:        "response.output_item.added",
			OutputIndex: 1,
			Item: &ResponsesOutput{
				Type:   "function_call",
				CallID: "call_1",
				Name:   "lookup",
			},
		},
		{
			Type:        "response.function_call_arguments.done",
			OutputIndex: 1,
			Arguments:   `{"q":"one"}`,
		},
		{
			Type:        "response.output_item.added",
			OutputIndex: 2,
			Item:        &ResponsesOutput{Type: "reasoning", ID: "rs_2", Status: "in_progress"},
		},
		{
			Type:        "response.output_item.done",
			OutputIndex: 2,
			Item: &ResponsesOutput{
				Type:             "reasoning",
				ID:               "rs_2",
				Status:           "completed",
				EncryptedContent: "enc-second",
			},
		},
		{Type: "response.completed", Response: &ResponsesResponse{ID: "resp_stream", Status: "completed"}},
	}

	var converted []AnthropicStreamEvent
	for i := range events {
		converted = append(converted, ResponsesEventToAnthropicEvents(&events[i], state)...)
	}

	var blockTypes []string
	var signatures []string
	for i, event := range converted {
		if event.Type == "content_block_start" && event.ContentBlock != nil {
			blockTypes = append(blockTypes, event.ContentBlock.Type)
		}
		if event.Type != "content_block_delta" || event.Delta == nil || event.Delta.Type != "signature_delta" {
			continue
		}
		signatures = append(signatures, event.Delta.Signature)
		require.Less(t, i+1, len(converted))
		assert.Equal(t, "content_block_stop", converted[i+1].Type)
		require.NotNil(t, event.Index)
		require.NotNil(t, converted[i+1].Index)
		assert.Equal(t, *event.Index, *converted[i+1].Index)
	}

	assert.Equal(t, []string{"thinking", "tool_use", "thinking"}, blockTypes)
	assert.Equal(t, []string{"enc-first", "enc-second"}, signatures)
}
