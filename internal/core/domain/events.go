package domain

type SearchQueryEvent struct {
	EventID   string
	Query     string
	UserID    string
	SessionID string
	DeviceID  string
	Locale    string
	Platform  string
	CreatedAt int64
}
