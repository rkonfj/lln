package tools

type CtxKey string

var (
	Provider      string = "provider"
	UniqueName    string = "uniqueName"
	StatusID      string = "statusID"
	UID           string = "uid"
	KeySession    CtxKey = "session"
	KeySessionUID CtxKey = "sessionUID"
)
