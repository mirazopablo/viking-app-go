package models

// TomorrowTaskDto represents the specific data required by n8n
// for the daily automation sync of tomorrow's appointments.
type TomorrowTaskDto struct {
	TaskID      string `json:"taskId"`
	EventID     string `json:"eventId"`
	ClientName  string `json:"clientName"`
	Phone       string `json:"phone"`
	ServiceType string `json:"serviceType"`
}
