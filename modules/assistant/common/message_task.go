package common

type MessageTask struct {
	SessionID  string
	CancelFunc func()
}
