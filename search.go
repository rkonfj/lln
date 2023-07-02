package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func search(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("type")
	value := r.URL.Query().Get("value")
	after := r.URL.Query().Get("after")
	sizeStr := r.URL.Query().Get("size")
	size := int64(20)
	if len(sizeStr) > 0 {
		var err error
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	var ss []*state.Status
	var more bool
	if t == "label" {
		ss, more = state.ListStatusByLabel(value, after, size)
	} else {
		ss = state.ListStatusByKeyword(value, after, size)
	}

	sessionUID := r.Context().Value(tools.KeySessionUID).(string)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s, sessionUID)
		if len(s.RefStatus) > 0 {
			prev := state.GetStatus(s.RefStatus)
			if prev != nil {
				status.RefStatus = castStatus(prev, sessionUID)
			}
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(L{V: ret, More: more})
}
