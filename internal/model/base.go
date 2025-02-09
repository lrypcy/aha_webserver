package model

type Status int
type ErrorCode int
type Config string

const (
	Pending Status = 0
	Started Status = 1
	Running Status = 2
	Failure Status = 3
	Success Status = 4
)
