package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/tools"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	tUser           string         = "/user/%s"
	tFollowUser     string         = "/user/%s/follow/%s"
	tFollowingUser  string         = "/user/%s/following/%s"
	UniqueNameRegex *regexp.Regexp = regexp.MustCompile(`^[\p{L}\d_]+$`)
)

type ModifiableUser struct {
	UniqueName string `json:"uniqueName"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	Bg         string `json:"bg"`
	Locale     string `json:"locale"`
	Bio        string `json:"bio"`
}

type User struct {
	ID           string    `json:"id"`
	UniqueName   string    `json:"uniqueName"`
	Name         string    `json:"name"`
	Picture      string    `json:"picture"`
	Bg           string    `json:"bg"`
	Email        string    `json:"email"`
	Locale       string    `json:"locale"`
	CreateTime   time.Time `json:"createTime"`
	Bio          string    `json:"bio"`
	VerifiedCode int64     `json:"verifiedCode"`
	ModRev       int64     `json:"-"`
}

// Modify apply new user props
func (u *User) Modify(mu ModifiableUser) error {
	if u.UniqueName == mu.UniqueName {
		mu.UniqueName = ""
	}
	key := stateKey(fmt.Sprintf(tUser, u.ID))

	ops := []clientv3.Op{}
	cmps := []clientv3.Cmp{clientv3.Compare(clientv3.ModRevision(key), "=", u.ModRev)}
	if len(mu.UniqueName) > 0 {
		oldUniqueNameKey := stateKey(fmt.Sprintf("/%s/%s", tools.UniqueName, u.UniqueName))
		uniqueNameKey := stateKey(fmt.Sprintf("/%s/%s", tools.UniqueName, mu.UniqueName))
		u.UniqueName = mu.UniqueName

		cmps = append(cmps, clientv3.Compare(clientv3.Version(uniqueNameKey), "=", 0))
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

	if len(mu.Bio) > 0 {
		u.Bio = mu.Bio
	}

	if len(mu.Bg) > 0 {
		u.Bg = mu.Bg
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
func (u *User) ListStatus(opts *tools.PaginationOptions) (ss []*Status, more bool) {
	return loadStatusByLinkerPagination(stateKey(fmt.Sprintf("/%s/status/", u.ID)), opts)
}

// FollowingBy determine if {uid} is following me
func (u *User) FollowingBy(user *ActUser) bool {
	if user == nil {
		return false
	}
	return Followed(user.ID, u.ID)
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

// Tweets tweet count
func (u *User) Tweets() int64 {
	resp, err := etcdClient.KV.Get(context.Background(),
		stateKey(fmt.Sprintf("/%s/status/", u.ID)),
		clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		logrus.Errorf("Tweets etcd error: %s", err)
		return 0
	}
	return resp.Count
}

func (u *User) SetVerified(code int64) error {
	key := stateKey(fmt.Sprintf(tUser, u.ID))
	u.VerifiedCode = code
	b, _ := json.Marshal(u)
	resp, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", u.ModRev)).
		Then(clientv3.OpPut(key, string(b))).Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return errors.New("try again later")
	}
	return nil
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
	ID           string `json:"id"`
	UniqueName   string `json:"uniqueName"`
	Name         string `json:"name"`
	Picture      string `json:"picture"`
	VerifiedCode int64  `json:"verifiedCode"`
}

type UserOptions struct {
	Name    string
	Picture string
	Email   string
	Locale  string
	Bio     string
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
		logrus.Errorf("cast user error: %s", err)
		return nil
	}
	u.ModRev = resp.Kvs[0].ModRevision
	return u
}

func NewUser(opts *UserOptions) (u *User, err error) {
	u = &User{
		Name:       opts.Name,
		Email:      opts.Email,
		Picture:    opts.Picture,
		Locale:     opts.Locale,
		Bio:        opts.Bio,
		CreateTime: time.Now(),
	}

	// generate a User ID
	u.ID = base58.Encode(xid.New().Bytes())

	userKey := stateKey(fmt.Sprintf(tUser, u.ID))
	emailKey := stateKey(fmt.Sprintf("/email/%s", u.Email))

	if UniqueNameRegex.MatchString(opts.Name) {
		u.UniqueName = opts.Name
		uniqueNameKey := stateKey(fmt.Sprintf("/uniqueName/%s", u.UniqueName))
		b, err := json.Marshal(u)
		if err != nil {
			return nil, err
		}
		resp, err := etcdClient.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(uniqueNameKey), "=", 0)).
			Then(clientv3.OpPut(userKey, string(b)),
				clientv3.OpPut(emailKey, userKey),
				clientv3.OpPut(uniqueNameKey, userKey)).
			Commit()
		if err != nil {
			logrus.Debugf("name as uniqueName error: %s, fallback to generate", err)
		} else if resp.Succeeded {
			return u, nil
		}
	}

	// generate a User UniqueName
	u.UniqueName = u.ID
	uniqueNameKey := stateKey(fmt.Sprintf("/uniqueName/%s", u.UniqueName))
	b, err := json.Marshal(u)
	if err != nil {
		return
	}
	resp, err := etcdClient.Txn(context.Background()).
		Then(clientv3.OpPut(userKey, string(b)),
			clientv3.OpPut(emailKey, userKey),
			clientv3.OpPut(uniqueNameKey, userKey)).
		Commit()
	if err != nil {
		return nil, err
	}
	if !resp.Succeeded {
		return nil, ErrTryAgainLater
	}
	return
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
