package history

import (
	"github.com/thomasbarrett/soren/internal/transcript"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
)

type History struct {
	records []transcript.Record
	t       *transcript.Transcript
}

func NewHistory(t *transcript.Transcript) *History {
	return &History{
		t:       t,
		records: []transcript.Record{},
	}
}

func (h *History) Messages() []openai.ChatCompletionMessageParamUnion {
	resp := make([]openai.ChatCompletionMessageParamUnion, len(h.records))

	for i, record := range h.records {
		resp[i] = record.Message
	}

	return resp
}

func (h *History) Add(msg openai.ChatCompletionMessageParamUnion) error {
	record := transcript.Record{
		Uuid:    uuid.New(),
		Message: msg,
	}

	h.records = append(h.records, record)

	return h.t.Write(record)
}

func (h *History) FindResponse(toolCallID string) *openai.ChatCompletionMessageParamUnion {
	for _, record := range h.records {
		msg := record.Message
		if msg.OfTool != nil && msg.OfTool.ToolCallID == toolCallID {
			return &msg
		}
	}

	return nil
}

func (h *History) Close() {
	h.t.Close()
}
