package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
)

func explore(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	user := currentSessionUser(r)
	ss, more := state.Recommendations(user, opts)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s, user)
		if s.Comments > 0 {
			meta, err := state.NewCommentsRecommandMeta(s.ID)
			if err != nil {
				logrus.Errorf("create comments recommand meta error: %s", err)
				continue
			}
			next := meta.Recommand()
			if next != nil {
				status.Next = castStatus(next, user)
			}
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(L{V: ret, More: more})
}

func exploreNewsProbe(w http.ResponseWriter, r *http.Request) {
	after, err := tools.URLQueryInt64(r, "after")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	user := currentSessionUser(r)
	json.NewEncoder(w).Encode(R{V: state.RecommendCount(user, after)})
}
