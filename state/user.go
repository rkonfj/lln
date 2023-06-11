package state

type User struct {
	UniqueName string
	Name       string
	Picture    string
	Email      string
}

func UserByEmail(email string) *User {
	return &User{}
}

func NewUser(email, name, picture string) *User {
	return &User{}
}
