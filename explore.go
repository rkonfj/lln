package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
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

type NewsProbeResponse struct {
	News        int              `json:"news"`
	CommentsMap map[string]int64 `json:"cm"`
	LikesMap    map[string]int64 `json:"lm"`
}

func exploreNewsProbe(w http.ResponseWriter, r *http.Request) {
	maxCreateRev, err := tools.URLQueryInt64(r, "max")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	minCreateRev, err := tools.URLQueryInt64(r, "min")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	user := currentSessionUser(r)
	probeResp := NewsProbeResponse{
		News:        state.RecommendCount(user, maxCreateRev),
		CommentsMap: state.CommentsMap(minCreateRev, maxCreateRev),
		LikesMap:    state.LikesMap(minCreateRev, maxCreateRev),
	}
	json.NewEncoder(w).Encode(R{V: probeResp})
}

func exploreStatusComment(w http.ResponseWriter, r *http.Request) {
	meta, err := state.NewCommentsRecommandMeta(chi.URLParam(r, tools.StatusID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	s := meta.Recommand()
	if s != nil {
		json.NewEncoder(w).Encode(R{V: castStatus(s, currentSessionUser(r))})
		return
	}
	json.NewEncoder(w).Encode(R{})
}
