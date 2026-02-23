package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusBlocked    TaskStatus = "blocked"
)

// TaskMetadata contains additional task information as unstructured JSON
type TaskMetadata map[string]interface{}

// Task represents a kanban task
type Task struct {
	ID          string       `json:"id"`
	Subject     string       `json:"subject"`
	Description string       `json:"description"`
	Status      TaskStatus   `json:"status"`
	Blocks      []string     `json:"blocks,omitempty"`
	BlockedBy   []string     `json:"blockedBy,omitempty"`
	Metadata    TaskMetadata `json:"metadata"`
}

// TaskManager manages tasks for a specific session
type TaskManager struct {
	sessionID string
	basePath  string
	mu        sync.RWMutex
}

// NewTaskManager creates a new task manager for the given session
func NewTaskManager(sessionID string) *TaskManager {
	return &TaskManager{
		sessionID: sessionID,
		basePath:  fmt.Sprintf(".soren/tasks/%s", sessionID),
	}
}

// ensurePath creates the task directory if it doesn't exist
func (tm *TaskManager) ensurePath() error {
	return os.MkdirAll(tm.basePath, 0755)
}

// getTaskFilePath returns the file path for a task
func (tm *TaskManager) getTaskFilePath(taskID string) string {
	return filepath.Join(tm.basePath, fmt.Sprintf("%s.json", taskID))
}

// CreateTask creates a new task
func (tm *TaskManager) CreateTask(task *Task) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if err := tm.ensurePath(); err != nil {
		return nil, fmt.Errorf("failed to create task directory: %w", err)
	}

	// Always generate a new ID, ignoring any provided ID
	task.ID = tm.nextID()

	// Set default status if not provided
	if task.Status == "" {
		task.Status = StatusTodo
	}

	// Initialize empty slices if nil
	if task.Blocks == nil {
		task.Blocks = []string{}
	}
	if task.BlockedBy == nil {
		task.BlockedBy = []string{}
	}

	return task, tm.saveTask(task)
}

// GetTask retrieves a task by ID
func (tm *TaskManager) GetTask(taskID string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	filePath := tm.getTaskFilePath(taskID)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task %s not found", taskID)
		}
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task data: %w", err)
	}

	return &task, nil
}

// ListTasks returns all tasks for the session, optionally filtered
func (tm *TaskManager) ListTasks(query *TaskQuery) ([]*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if err := tm.ensurePath(); err != nil {
		return nil, fmt.Errorf("failed to create task directory: %w", err)
	}

	entries, err := os.ReadDir(tm.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read task directory: %w", err)
	}

	var tasks []*Task
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		taskID := filepath.Base(entry.Name())
		taskID = taskID[:len(taskID)-5] // Remove .json extension

		task, err := tm.GetTask(taskID)
		if err != nil {
			// Skip corrupted tasks but don't fail entirely
			continue
		}

		// Apply query if provided
		if query == nil || query.matches(task) {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// UpdateTask updates an existing task using a function
func (tm *TaskManager) UpdateTask(taskID string, updates func(*Task) error) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, err := tm.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if err := updates(task); err != nil {
		return nil, fmt.Errorf("failed to apply updates: %w", err)
	}

	return task, tm.saveTask(task)
}

// DeleteTask removes a task
func (tm *TaskManager) DeleteTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	filePath := tm.getTaskFilePath(taskID)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("task %s not found", taskID)
		}
		return fmt.Errorf("failed to delete task file: %w", err)
	}

	return nil
}

// saveTask saves a task to disk
func (tm *TaskManager) saveTask(task *Task) error {
	filePath := tm.getTaskFilePath(task.ID)
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// nextID generates the next sequential task ID
func (tm *TaskManager) nextID() string {
	// Find highest existing ID
	maxID := 0
	if entries, err := os.ReadDir(tm.basePath); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			name := entry.Name()
			idStr := name[:len(name)-5] // Remove .json extension

			if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
				maxID = id
			}
		}
	}

	return fmt.Sprintf("%d", maxID+1)
}
