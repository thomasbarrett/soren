package transcript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
)

type Transcript struct {
	file *os.File
	mu   sync.Mutex
}

type Record struct {
	Uuid    uuid.UUID                              `json:"uuid"`
	Message openai.ChatCompletionMessageParamUnion `json:"message"`
}

func NewTranscript(filePath string) (*Transcript, error) {
	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return nil, err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return &Transcript{file: file}, nil
}

func (t *Transcript) Write(r Record) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	line, err := json.Marshal(r)
	if err != nil {
		return err
	}

	_, err = t.file.Write(append(line, '\n'))
	return err
}

func (t *Transcript) Close() error {
	return t.file.Close()
}
