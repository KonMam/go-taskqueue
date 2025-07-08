package model

type Task struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Result  int    `json:"result"`
	Retries int    `json:"retries"`
	// Add type e.g. compute, email, etc. for actual processing
}
