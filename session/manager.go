package session

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/state"
	"github.com/rs/xid"
)

type Session struct {
	ApiKey     string `json:"apiKey"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Picture    string `json:"picture"`
}

func (s *Session) ToUser() *state.ActUser {
	return &state.ActUser{
		ID:         s.ID,
		Name:       s.Name,
		UniqueName: s.UniqueName,
		Picture:    s.Picture,
	}
}

type SessionManager interface {
	Create(*Session) error
	Load(string) *Session
}

type MemorySessionManger struct {
	lock    sync.RWMutex
	session map[string]*Session
}

func NewSessionManager() *MemorySessionManger {
	return &MemorySessionManger{
		session: make(map[string]*Session),
	}
}

func (sm *MemorySessionManger) Create(s *Session) error {
	b := make([]byte, 16)
	rand.Reader.Read(b)
	s.ApiKey = fmt.Sprintf("sk-%s", base58.Encode(append(b, xid.New().Bytes()...)))
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.session[s.ApiKey] = s
	return nil
}

func (sm *MemorySessionManger) Load(key string) *Session {
	return sm.session[key]
}
