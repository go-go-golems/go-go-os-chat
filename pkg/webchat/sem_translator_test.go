package webchat

import (
	"encoding/json"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func decodeSemEvent(t *testing.T, frame []byte) map[string]any {
	t.Helper()
	var env map[string]any
	require.NoError(t, json.Unmarshal(frame, &env))
	require.Equal(t, true, env["sem"])
	ev, ok := env["event"].(map[string]any)
	require.True(t, ok)
	return ev
}

func TestSemanticEventsFromEvent_UsesStableLLMIDs(t *testing.T) {
	meta := events.EventMetadata{
		SessionID:   "sess-1",
		InferenceID: "inf-1",
		TurnID:      "turn-1",
	}

	startFrames := SemanticEventsFromEvent(events.NewTextSegmentStartedEvent(meta, events.Correlation{}, "assistant"))
	require.Len(t, startFrames, 1)
	start := decodeSemEvent(t, startFrames[0])
	require.Equal(t, "llm.start", start["type"])

	deltaFrames := SemanticEventsFromEvent(events.NewTextDeltaEvent(meta, events.Correlation{}, "Hello", "Hello", 1))
	require.Len(t, deltaFrames, 1)
	delta := decodeSemEvent(t, deltaFrames[0])
	require.Equal(t, "llm.delta", delta["type"])

	finalFrames := SemanticEventsFromEvent(events.NewTextSegmentFinishedEvent(meta, events.Correlation{}, "Hello", "stop"))
	require.Len(t, finalFrames, 1)
	final := decodeSemEvent(t, finalFrames[0])
	require.Equal(t, "llm.final", final["type"])

	wantID := "llm-inf-1"
	require.Equal(t, wantID, start["id"])
	require.Equal(t, wantID, delta["id"])
	require.Equal(t, wantID, final["id"])

	startData, ok := start["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "assistant", startData["role"])

	deltaData, ok := delta["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello", deltaData["delta"])

	finalData, ok := final["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello", finalData["text"])
}

func TestSemanticEventsFromEvent_DropsWhitespaceOnlyDeltas(t *testing.T) {
	meta := events.EventMetadata{
		SessionID:   "sess-ws",
		InferenceID: "inf-ws",
		TurnID:      "turn-ws",
	}

	startFrames := SemanticEventsFromEvent(events.NewTextSegmentStartedEvent(meta, events.Correlation{}, "assistant"))
	require.Len(t, startFrames, 1)
	start := decodeSemEvent(t, startFrames[0])
	wantID := start["id"].(string)

	frames := SemanticEventsFromEvent(events.NewTextDeltaEvent(meta, events.Correlation{}, " \n\t ", " \n\t ", 1))
	require.Len(t, frames, 0)

	deltaFrames := SemanticEventsFromEvent(events.NewTextDeltaEvent(meta, events.Correlation{}, "Hello", "Hello", 1))
	require.Len(t, deltaFrames, 1)
	delta := decodeSemEvent(t, deltaFrames[0])
	require.Equal(t, "llm.delta", delta["type"])
	require.Equal(t, wantID, delta["id"])
}

func TestSemanticEventsFromEvent_PrefersExplicitUUIDMessageID(t *testing.T) {
	msgID := uuid.New()
	meta := events.EventMetadata{
		ID:          msgID,
		SessionID:   "sess-2",
		InferenceID: "inf-2",
		TurnID:      "turn-2",
	}

	frames := SemanticEventsFromEvent(events.NewTextDeltaEvent(meta, events.Correlation{}, "a", "a", 1))
	require.Len(t, frames, 1)
	ev := decodeSemEvent(t, frames[0])
	require.Equal(t, msgID.String(), ev["id"])

	data, ok := ev["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "a", data["delta"])
}

func TestSemanticEventsFromEvent_CalcToolResultGetsCustomKind(t *testing.T) {
	meta := events.EventMetadata{SessionID: "sess-3", InferenceID: "inf-3", TurnID: "turn-3"}

	callFrames := SemanticEventsFromEvent(events.NewToolCallRequestedEvent(meta, events.Correlation{}, "tc-1", "calc", `{"expression":"1+1"}`))
	require.Len(t, callFrames, 1)
	callEv := decodeSemEvent(t, callFrames[0])
	require.Equal(t, "tool.start", callEv["type"])
	callData, ok := callEv["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "calc", callData["name"])

	resultFrames := SemanticEventsFromEvent(events.NewToolResultReadyEvent(meta, events.Correlation{}, "tc-1", "calc", "2", "completed"))
	require.Len(t, resultFrames, 2)
	resEv := decodeSemEvent(t, resultFrames[0])
	doneEv := decodeSemEvent(t, resultFrames[1])
	require.Equal(t, "tool.result", resEv["type"])
	resData, ok := resEv["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "calc_result", resData["customKind"])
	require.Equal(t, "tool.done", doneEv["type"])
}

func TestSemanticEventsFromEvent_ReasoningSummaryMapsToThinkingSummary(t *testing.T) {
	meta := events.EventMetadata{
		SessionID:   "sess-summary",
		InferenceID: "inf-summary",
		TurnID:      "turn-summary",
	}

	frames := SemanticEventsFromEvent(events.NewInfoEvent(meta, "reasoning-summary", map[string]any{
		"text": "final reasoning summary",
	}))
	require.Len(t, frames, 1)

	ev := decodeSemEvent(t, frames[0])
	require.Equal(t, "llm.thinking.summary", ev["type"])
	require.Equal(t, "llm-inf-summary:thinking", ev["id"])

	data, ok := ev["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "final reasoning summary", data["text"])
}
