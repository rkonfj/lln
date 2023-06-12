package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/decred/base58"
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

type ActUser struct {
	ID         string `json:"id"`
	UniqueName string `json:"uniqueName"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
}

type UserOptions struct {
	Name    string
	Picture string
	Email   string
	Locale  string
}

func UserByEmail(email string) *User {
	return castUser(getPointerValue(stateKey(fmt.Sprintf("/email/%s", email))))
}

func UserByUniqueName(uniqueName string) *User {
	return castUser(getPointerValue(stateKey(fmt.Sprintf("/uniqueName/%s", uniqueName))))
}

func castUser(resp *clientv3.GetResponse, err error) *User {
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
	u.ID = base58.Encode(xid.New().Bytes())

	// generate a User UniqueName
	u.UniqueName = u.ID

	b, err := json.Marshal(u)
	if err != nil {
		logrus.Debug(err)
		return nil
	}

	userKey := stateKey(fmt.Sprintf("/user/%s", u.ID))
	emailKey := stateKey(fmt.Sprintf("/email/%s", u.Email))
	uniqueNameKey := stateKey(fmt.Sprintf("/uniqueName/%s", u.UniqueName))
	_, err = etcdClient.Txn(context.Background()).
		Then(
			clientv3.OpPut(userKey, string(b)),
			clientv3.OpPut(emailKey, userKey),
			clientv3.OpPut(uniqueNameKey, userKey),
		).
		Commit()
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return u
}

func LikeUser(user *ActUser, uniqueName string) error {
	targetUser := UserByUniqueName(uniqueName)
	if targetUser == nil {
		return errors.New("not found")
	}
	userLikeSetKey := stateKey(fmt.Sprintf("/user/%s/like/%s", targetUser.ID, user.ID))
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	_, err = etcdClient.KV.Put(context.Background(), userLikeSetKey, string(b))
	return err
}
