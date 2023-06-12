package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type User struct {
	ID         string `json:"id"`
	UniqueName string `json:"uniqueName"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	Email      string `json:"email"`
	Locale     string `json:"locale"`
}

type UserOptions struct {
	Name    string
	Picture string
	Email   string
	Locale  string
}

func UserByEmail(email string) *User {
	// email pointer to user test
	resp, err := getPointerValue(stateKey(fmt.Sprintf("/email/%s", email)))
	if err != nil {
		logrus.Debug(err)
		return nil
	}

	if len(resp.Kvs) == 0 {
		return nil
	}

	u := &User{}
	err = json.Unmarshal(resp.Kvs[0].Value, u)
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return u
}

func NewUser(opts *UserOptions) *User {
	u := &User{
		Name:    opts.Name,
		Email:   opts.Email,
		Picture: opts.Picture,
		Locale:  opts.Locale,
	}

	// generate a User ID
	u.ID = xid.New().String()

	// generate a User UniqueName
	u.UniqueName = u.ID

	b, err := json.Marshal(u)
	if err != nil {
		logrus.Debug(err)
		return nil
	}

	userKey := stateKey(fmt.Sprintf("/user/%s", u.ID))
	emailKey := stateKey(fmt.Sprintf("/email/%s", u.Email))
	_, err = etcdClient.Txn(context.Background()).
		Then(clientv3.OpPut(userKey, string(b)), clientv3.OpPut(emailKey, userKey)).
		Commit()
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return u
}
