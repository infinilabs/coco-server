package common

type MessageTask struct {
	SessionID string
	// Deprecated
	TaskID string

	CancelFunc func()
}
