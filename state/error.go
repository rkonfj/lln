package state

import "errors"

var (
	ErrStatusNotFound       = errors.New("status not found")
	ErrStatusQuotes   error = errors.New("there are quotes")
	ErrTryAgainLater  error = errors.New("txn failed. try again later")
)
