package session

type Session struct {
	ApiKey     string `json:"apiKey"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Picture    string `json:"picture"`
}

func Create(email, name, picture string) (*Session, error) {
	return &Session{
		ApiKey:     "",
		Name:       name,
		UniqueName: "",
		Picture:    picture,
	}, nil
}
