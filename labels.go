package main

import (
	"encoding/json"
	"net/http"

	"github.com/rkonfj/lln/state"
)

func labels(w http.ResponseWriter, r *http.Request) {
	labels := state.GetLabels(r.URL.Query().Get("prefix"))
	json.NewEncoder(w).Encode(labels)
}
