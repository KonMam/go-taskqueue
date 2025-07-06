package model

type Task struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Result int    `json:"result"`
}
