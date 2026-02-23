package tasks

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskValidationError represents a validation error
type TaskValidationError struct {
	Field   string
	Message string
}

func (e TaskValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidateTask validates a task's data
func ValidateTask(task *Task) []TaskValidationError {
	var errors []TaskValidationError

	// Validate subject
	if strings.TrimSpace(task.Subject) == "" {
		errors = append(errors, TaskValidationError{
			Field:   "subject",
			Message: "subject cannot be empty",
		})
	} else if len(task.Subject) > 200 {
		errors = append(errors, TaskValidationError{
			Field:   "subject",
			Message: "subject cannot exceed 200 characters",
		})
	}

	// Validate description
	if len(task.Description) > 5000 {
		errors = append(errors, TaskValidationError{
			Field:   "description",
			Message: "description cannot exceed 5000 characters",
		})
	}

	// Validate status
	if !isValidStatus(task.Status) {
		errors = append(errors, TaskValidationError{
			Field:   "status",
			Message: fmt.Sprintf("invalid status: %s", task.Status),
		})
	}

	// Validate task ID format
	if task.ID != "" && !isValidTaskID(task.ID) {
		errors = append(errors, TaskValidationError{
			Field:   "id",
			Message: "task ID must be a positive integer",
		})
	}

	// Validate blocking relationships
	if containsDuplicates(task.Blocks) {
		errors = append(errors, TaskValidationError{
			Field:   "blocks",
			Message: "blocks list contains duplicates",
		})
	}

	if containsDuplicates(task.BlockedBy) {
		errors = append(errors, TaskValidationError{
			Field:   "blockedBy",
			Message: "blockedBy list contains duplicates",
		})
	}

	// Check for self-blocking
	for _, id := range task.Blocks {
		if id == task.ID {
			errors = append(errors, TaskValidationError{
				Field:   "blocks",
				Message: "task cannot block itself",
			})
		}
	}

	for _, id := range task.BlockedBy {
		if id == task.ID {
			errors = append(errors, TaskValidationError{
				Field:   "blockedBy",
				Message: "task cannot be blocked by itself",
			})
		}
	}

	return errors
}

// isValidStatus checks if the status is valid
func isValidStatus(status TaskStatus) bool {
	switch status {
	case StatusTodo, StatusInProgress, StatusDone, StatusBlocked:
		return true
	default:
		return false
	}
}

// isValidTaskID checks if the task ID is valid (positive integer as string)
func isValidTaskID(id string) bool {
	validID := regexp.MustCompile(`^[1-9]\d*$`)
	return validID.MatchString(id)
}

// containsDuplicates checks if a string slice contains duplicates
func containsDuplicates(slice []string) bool {
	seen := make(map[string]bool)
	for _, item := range slice {
		if seen[item] {
			return true
		}
		seen[item] = true
	}
	return false
}

// SanitizeTask sanitizes task data
func SanitizeTask(task *Task) {
	// Trim whitespace from subject and description
	task.Subject = strings.TrimSpace(task.Subject)
	task.Description = strings.TrimSpace(task.Description)

	// Remove duplicates from blocks and blockedBy
	task.Blocks = removeDuplicateStrings(task.Blocks)
	task.BlockedBy = removeDuplicateStrings(task.BlockedBy)

	// Initialize metadata if nil
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
}

// removeEmptyStrings removes empty strings from a slice
func removeEmptyStrings(slice []string) []string {
	var result []string
	for _, s := range slice {
		if strings.TrimSpace(s) != "" {
			result = append(result, strings.TrimSpace(s))
		}
	}
	return result
}

// removeDuplicateStrings removes duplicate strings from a slice
func removeDuplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ValidateStatusTransition checks if a status transition is valid
func ValidateStatusTransition(from, to TaskStatus) error {
	// Define valid transitions
	validTransitions := map[TaskStatus][]TaskStatus{
		StatusTodo:       {StatusInProgress, StatusBlocked, StatusDone},
		StatusInProgress: {StatusTodo, StatusDone, StatusBlocked},
		StatusBlocked:    {StatusTodo, StatusInProgress},
		StatusDone:       {StatusTodo, StatusInProgress}, // Allow reopening completed tasks
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("unknown status: %s", from)
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == to {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", from, to)
}
