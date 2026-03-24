package models

type TicketTaskMsg struct {
	OrderID    int64 `json:"orderId"`
	ActivityID int64 `json:"activityId"`
	Need       int   `json:"need"`
}
