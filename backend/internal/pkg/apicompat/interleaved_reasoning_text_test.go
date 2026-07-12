package apicompat

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicToResponsesPreservesAdjacentTextSeparatorsAcrossReasoning(t *testing.T) {
	var req AnthropicRequest
	require.NoError(t, json.Unmarshal([]byte(`{
		"model":"gpt-5.6",
		"messages":[{"role":"assistant","content":[
			{"type":"text","text":"before one"},
			{"type":"text","text":"before two"},
			{"type":"thinking","thinking":"plan","signature":"enc-plan"},
			{"type":"text","text":"after one"},
			{"type":"text","text":"after two"}
		]}]
	}`), &req))

	resp, err := AnthropicToResponses(&req)
	require.NoError(t, err)
	var items []ResponsesInputItem
	require.NoError(t, json.Unmarshal(resp.Input, &items))
	require.Len(t, items, 3)
	assert.Equal(t, "before one\n\nbefore two", responseItemText(t, items[0]))
	assert.Equal(t, "reasoning", items[1].Type)
	assert.Equal(t, "after one\n\nafter two", responseItemText(t, items[2]))
}

func responseItemText(t *testing.T, item ResponsesInputItem) string {
	t.Helper()
	var parts []ResponsesContentPart
	require.NoError(t, json.Unmarshal(item.Content, &parts))
	require.Len(t, parts, 1)
	return parts[0].Text
}
