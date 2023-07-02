package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func explore(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	size := int64(20)
	var err error
	if len(sizeStr) != 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	var user *state.ActUser
	if r.Context().Value(tools.KeySession) != nil {
		user = r.Context().Value(tools.KeySession).(*state.Session).ToUser()
	}

	sessionUID := r.Context().Value(tools.KeySessionUID).(string)
	ss, more := state.Recommendations(user, r.URL.Query().Get("after"), size)
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
