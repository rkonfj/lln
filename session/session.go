package session

import "github.com/rkonfj/lln/state"

var defaultSessionManager = NewSessionManager()

func Create(opts *state.UserOptions) (*Session, error) {
	u := state.UserByEmail(opts.Email)
	if u == nil {
		u = state.NewUser(opts)
	}
	s := &Session{
		Name:       u.Name,
		UniqueName: u.UniqueName,
		Picture:    u.Picture,
	}
	defaultSessionManager.Create(s)
	return s, nil
}
