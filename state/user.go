package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/util"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	tUser          string = "/user/%s"
	tFollowUser    string = "/user/%s/follow/%s"
	tFollowingUser string = "/user/%s/following/%s"
)

type ModifiableUser struct {
	UniqueName string `json:"uniqueName"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	Locale     string `json:"locale"`
}

type User struct {
	ID         string    `json:"id"`
	UniqueName string    `json:"uniqueName"`
	Name       string    `json:"name"`
	Picture    string    `json:"picture"`
	Email      string    `json:"email"`
	Locale     string    `json:"locale"`
	CreateTime time.Time `json:"createTime"`
}

// Modify apply new user props
func (u *User) Modify(mu ModifiableUser) error {
	if u.UniqueName == mu.UniqueName {
		mu.UniqueName = ""
	}
	key := stateKey(fmt.Sprintf(tUser, u.ID))
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return err
	}

	ops := []clientv3.Op{}
	cmps := []clientv3.Cmp{clientv3.Compare(clientv3.ModRevision(key), "=", resp.Kvs[0].ModRevision)}
	if len(mu.UniqueName) > 0 {
		oldUniqueNameKey := stateKey(fmt.Sprintf("/%s/%s", util.UniqueName, u.UniqueName))
		uniqueNameKey := stateKey(fmt.Sprintf("/%s/%s", util.UniqueName, mu.UniqueName))
		u.UniqueName = mu.UniqueName

		cmps = append(cmps, clientv3.Compare(clientv3.Version(mu.UniqueName), "=", 0))
		ops = append(ops, clientv3.OpDelete(oldUniqueNameKey))
		ops = append(ops, clientv3.OpPut(uniqueNameKey, key))
	}

	if len(mu.Name) > 0 {
		u.Name = mu.Name
	}

	if len(mu.Picture) > 0 {
		u.Picture = mu.Picture
	}

	if len(mu.Locale) > 0 {
		u.Locale = mu.Locale
	}

	b, err := json.Marshal(u)
	if err != nil {
		return err
	}
	ops = append(ops, clientv3.OpPut(key, string(b)))

	txnResp, err := etcdClient.Txn(context.Background()).If(cmps...).Then(ops...).Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		return errors.New("failed. unique name already exists")
	}
	return nil
}

// ListStatus list user all status
func (u *User) ListStatus(after string, size int64) (ss []*Status) {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/%s/status/", u.ID)), after, size)
}

// FollowingBy determine if {uid} is following me
func (u *User) FollowingBy(uid string) bool {
	return Followed(uid, u.ID)
}

// Followers follower count
func (u *User) Followers() int64 {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf(tFollowUser, u.ID, "")),
		clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		logrus.Errorf("FollowingBy etcd error: %s", err)
		return 0
	}
	return resp.Count
}

// Followings following count
func (u *User) Followings() int64 {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf(tFollowingUser, u.ID, "")),
		clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		logrus.Errorf("Followings etcd error: %s", err)
		return 0
	}
	return resp.Count
}

func Followed(u1, u2 string) bool {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf(tFollowUser, u2, u1)), clientv3.WithCountOnly())
	if err != nil {
		logrus.Errorf("FollowingBy error: %s", err)
		return false
	}
	return resp.Count == 1
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

func UserByID(userID string) *User {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf(tUser, userID)))
	return castUser(resp, err)
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
		Name:       opts.Name,
		Email:      opts.Email,
		Picture:    opts.Picture,
		Locale:     opts.Locale,
		CreateTime: time.Now(),
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

	userKey := stateKey(fmt.Sprintf(tUser, u.ID))
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

func FollowUser(user *ActUser, uniqueName string) error {
	targetUser := UserByUniqueName(uniqueName)
	if targetUser == nil {
		return errors.New("not found")
	}
	followUserKey := stateKey(fmt.Sprintf(tFollowUser, targetUser.ID, user.ID))
	followingUserKey := stateKey(fmt.Sprintf(tFollowingUser, user.ID, targetUser.ID))
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}

	delOps := []clientv3.Op{clientv3.OpDelete(followUserKey), clientv3.OpDelete(followingUserKey)}
	newOps := []clientv3.Op{clientv3.OpPut(followUserKey, string(b)),
		clientv3.OpPut(followingUserKey, stateKey(fmt.Sprintf(tUser, targetUser.ID)))}
	newOps = append(newOps, newMessageOps(MsgOptions{
		from:     user,
		toUID:    targetUser.ID,
		msgType:  MsgTypeFollow,
		targetID: targetUser.ID,
	})...)
	_, err = etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(followUserKey), ">", 0)).
		Then(delOps...).Else(newOps...).Commit()
	return err
}
