package common

type MessageTask struct {
	SessionID  string
	CancelFunc func()
	// CancelLang is the user's UI language (e.g. "zh-CN", "en"), set by the
	// cancel API right before invoking CancelFunc. The async processor reads
	// this from the InflightMessages map so it can persist a localized
	// "task cancelled" message into the reply — the backend has no i18n
	// system, so the frontend supplies the language at cancel time.
	CancelLang string
}
