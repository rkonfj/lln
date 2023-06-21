package main

import (
	"net/http"
)

func listMessages(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
