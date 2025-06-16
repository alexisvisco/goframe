package temporalutil

import "strings"

// IsScheduleAlreadyExistError checks if the error indicates that a schedule with the same ID already exists.
func IsScheduleAlreadyExistError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "schedule with this ID is already registered")
}
