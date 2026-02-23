package tasks

// TaskQuery represents criteria for querying tasks
type TaskQuery struct {
	Status   *TaskStatus            `json:"status,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// matchesQuery checks if a task matches the given query
func (query *TaskQuery) matches(task *Task) bool {
	if query == nil {
		return true
	}

	// Status filter
	if query.Status != nil && task.Status != *query.Status {
		return false
	}

	// Metadata filter - check if all query metadata key-value pairs match
	if len(query.Metadata) > 0 {
		if task.Metadata == nil {
			return false
		}
		for key, expectedValue := range query.Metadata {
			actualValue, exists := task.Metadata[key]
			if !exists || actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}
