package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func labels(w http.ResponseWriter, r *http.Request) {
	size, err := tools.URLQueryInt64Default(r, "size", 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	labels := state.GetLabels(r.URL.Query().Get("prefix"), size)
	json.NewEncoder(w).Encode(L{V: labels})
}
