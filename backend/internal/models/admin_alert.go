package models

type AdminAlert struct {
	Alert
	UserID    string  `json:"user_id"`
	UserEmail *string `json:"user_email"`
}

type AdminAlertFilter struct {
	Symbol string
	Status string
	UserID string
	Limit  int
	Offset int
}

type AdminAlertPage struct {
	Alerts []AdminAlert
	Page   int
	Limit  int
	Total  int64
}

type DisableAlertRequest struct {
	Reason string `json:"reason"`
}
