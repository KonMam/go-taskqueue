package queue

import "go-taskqueue/model"

var Tasks = make(chan model.Task, 100)
