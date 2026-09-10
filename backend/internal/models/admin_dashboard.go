package models

type AdminDashboard struct {
	Users  AdminUserCounts  `json:"users"`
	Alerts AdminAlertCounts `json:"alerts"`
	Line   AdminLineCounts  `json:"line"`
}

type AdminUserCounts struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Disabled int64 `json:"disabled"`
}

type AdminAlertCounts struct {
	Active    int64 `json:"active"`
	Triggered int64 `json:"triggered"`
}

type AdminLineCounts struct {
	ConnectedUsers int64 `json:"connected_users"`
	FailedMessages int64 `json:"failed_messages"`
}
