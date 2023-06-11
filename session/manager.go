package session

import (
	"crypto/rand"
	"sync"

	"github.com/decred/base58"
)

type Session struct {
	ApiKey     string `json:"apiKey"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Picture    string `json:"picture"`
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
	b := make([]byte, 32)
	rand.Reader.Read(b)
	s.ApiKey = base58.Encode(b)
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.session[s.ApiKey] = s
	return nil
}

func (sm *MemorySessionManger) Load(key string) *Session {
	return sm.session[key]
}
