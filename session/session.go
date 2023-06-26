package session

import "github.com/rkonfj/lln/state"

var DefaultSessionManager = NewSessionManager()

func Create(opts *state.UserOptions) (*Session, error) {
	u := state.UserByEmail(opts.Email)
	if u == nil {
		u = state.NewUser(opts)
	}
	s := &Session{
		ID:         u.ID,
		Name:       u.Name,
		UniqueName: u.UniqueName,
		Picture:    u.Picture,
		Locale:     u.Locale,
	}
	DefaultSessionManager.Create(s)
	return s, nil
}
