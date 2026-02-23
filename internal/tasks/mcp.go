package tasks

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateTaskInput represents the input for creating a task
type CreateTaskInput struct {
	Subject     string                 `json:"subject"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Blocks      []string               `json:"blocks,omitempty"`
	BlockedBy   []string               `json:"blockedBy,omitempty"`
}

type TaskOutput struct {
	Task   *Task  `json:"task,omitempty"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type TaskListOutput struct {
	Tasks  []*Task `json:"tasks,omitempty"`
	Count  int     `json:"count"`
	Result string  `json:"result,omitempty"`
	Error  string  `json:"error,omitempty"`
}

// RegisterCreateTask registers the CreateTask MCP tool
func (tm *TaskManager) RegisterCreateTask(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "CreateTask",
		Description: "Create a new kanban task with the specified properties. Task ID is auto-generated.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args CreateTaskInput) (*mcp.CallToolResult, TaskOutput, error) {
		task := &Task{
			Subject:     args.Subject,
			Description: args.Description,
			Blocks:      args.Blocks,
			BlockedBy:   args.BlockedBy,
			Metadata:    args.Metadata,
		}

		// Set status
		if args.Status != "" {
			task.Status = TaskStatus(args.Status)
		}

		// Initialize metadata if nil
		if task.Metadata == nil {
			task.Metadata = make(map[string]interface{})
		}

		createdTask, err := tm.CreateTask(task)
		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		result := fmt.Sprintf("Created task #%s: %s", createdTask.ID, createdTask.Subject)
		return nil, TaskOutput{Task: createdTask, Result: result}, nil
	})
}

// GetTaskInput represents the input for getting a task
type GetTaskInput struct {
	ID string `json:"id"`
}

// RegisterGetTask registers the GetTask MCP tool
func (tm *TaskManager) RegisterGetTask(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "GetTask",
		Description: "Retrieve a specific task by its ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GetTaskInput) (*mcp.CallToolResult, TaskOutput, error) {
		task, err := tm.GetTask(args.ID)
		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		return nil, TaskOutput{Task: task, Result: fmt.Sprintf("Retrieved task #%s", args.ID)}, nil
	})
}

// ListTasksInput represents the input for listing tasks
type ListTasksInput struct {
	Status   string                 `json:"status,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterListTasks registers the ListTasks MCP tool
func (tm *TaskManager) RegisterListTasks(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ListTasks",
		Description: "List all tasks, optionally filtered by status, type, priority, or tags.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ListTasksInput) (*mcp.CallToolResult, TaskListOutput, error) {
		var query *TaskQuery
		if args.Status != "" || len(args.Metadata) > 0 {
			query = &TaskQuery{
				Metadata: args.Metadata,
			}

			if args.Status != "" {
				status := TaskStatus(args.Status)
				query.Status = &status
			}
		}

		taskList, err := tm.ListTasks(query)
		if err != nil {
			return nil, TaskListOutput{Error: err.Error()}, err
		}

		result := fmt.Sprintf("Found %d tasks", len(taskList))
		return nil, TaskListOutput{Tasks: taskList, Count: len(taskList), Result: result}, nil
	})
}

// UpdateTaskInput represents the input for updating a task
type UpdateTaskInput struct {
	ID          string                 `json:"id"`
	Subject     string                 `json:"subject,omitempty"`
	Description string                 `json:"description,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Blocks      []string               `json:"blocks,omitempty"`
	BlockedBy   []string               `json:"blockedBy,omitempty"`
}

// RegisterUpdateTask registers the UpdateTask MCP tool
func (tm *TaskManager) RegisterUpdateTask(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "UpdateTask",
		Description: "Update an existing task. Only provided fields will be updated, others remain unchanged.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args UpdateTaskInput) (*mcp.CallToolResult, TaskOutput, error) {
		updatedTask, err := tm.UpdateTask(args.ID, func(task *Task) error {
			if args.Subject != "" {
				task.Subject = args.Subject
			}
			if args.Description != "" {
				task.Description = args.Description
			}
			if args.Status != "" {
				task.Status = TaskStatus(args.Status)
			}
			if len(args.Metadata) > 0 {
				if task.Metadata == nil {
					task.Metadata = make(map[string]interface{})
				}
				for k, v := range args.Metadata {
					task.Metadata[k] = v
				}
			}
			if args.Blocks != nil {
				task.Blocks = args.Blocks
			}
			if args.BlockedBy != nil {
				task.BlockedBy = args.BlockedBy
			}
			return nil
		})

		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		result := fmt.Sprintf("Updated task #%s: %s", updatedTask.ID, updatedTask.Subject)
		return nil, TaskOutput{Task: updatedTask, Result: result}, nil
	})
}

// DeleteTaskInput represents the input for deleting a task
type DeleteTaskInput struct {
	ID string `json:"id"`
}

// RegisterDeleteTask registers the DeleteTask MCP tool
func (tm *TaskManager) RegisterDeleteTask(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "DeleteTask",
		Description: "Delete a task by its ID. This action cannot be undone.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args DeleteTaskInput) (*mcp.CallToolResult, TaskOutput, error) {
		err := tm.DeleteTask(args.ID)
		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		result := fmt.Sprintf("Deleted task #%s", args.ID)
		return nil, TaskOutput{Result: result}, nil
	})
}

// RegisterTaskTools registers all task-related MCP tools
func (tm *TaskManager) Register(server *mcp.Server) {
	tm.RegisterCreateTask(server)
	tm.RegisterGetTask(server)
	tm.RegisterListTasks(server)
	tm.RegisterUpdateTask(server)
	tm.RegisterDeleteTask(server)
}
