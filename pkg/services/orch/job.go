package orch

import "context"

type Job struct {
	ID      string
	jobFunc func(ctx context.Context) error
}

type JobFunc func(ctx context.Context) error

func NewJob(id string, jobFunc JobFunc) *Job {
	return &Job{
		ID:      id,
		jobFunc: jobFunc,
	}
}
