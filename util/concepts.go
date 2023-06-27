package util

type CtxKey string

var (
	Provider      string = "provider"
	UniqueName    string = "uniqueName"
	StatusID      string = "statusID"
	KeySession    CtxKey = "session"
	KeySessionUID CtxKey = "sessionUID"
)
