package main

import (
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func currentSessionUser(r *http.Request) (user *state.ActUser) {
	if r.Context().Value(tools.KeySession) != nil {
		user = r.Context().Value(tools.KeySession).(*state.Session).ToUser()
	}
	return
}

func castStatus(s *state.Status, sessionUser *state.ActUser) *Status {
	var liked, bookmarked, followed bool
	if sessionUser != nil {
		liked = state.Liked(s.ID, sessionUser.ID)
		bookmarked = state.Bookmarked(s.ID, sessionUser.ID)
		followed = state.Followed(sessionUser.ID, s.User.ID)
	}
	return &Status{
		ID:         s.ID,
		Content:    s.Content,
		User:       s.User,
		CreateRev:  s.CreateRev,
		CreateTime: s.CreateTime,
		Comments:   s.Comments,
		Views:      s.Views,
		LikeCount:  s.LikeCount,
		Bookmarks:  s.Bookmarks,
		Disabled:   s.Disabled,
		Liked:      liked,
		Bookmarked: bookmarked,
		Followed:   followed,
	}
}
