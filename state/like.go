package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var statusLikeKey func(statusID, uid string) string = func(statusID, uid string) string {
	return stateKey(fmt.Sprintf("/like/status/%s/%s", statusID, uid))
}

var userLikeKey func(statusID, uid string) string = func(statusID, uid string) string {
	return stateKey(fmt.Sprintf("/like/%s/status/%s", uid, statusID))
}

func Liked(statusID, uid string) bool {
	key := statusLikeKey(statusID, uid)
	resp, err := etcdClient.KV.Get(context.Background(), key, clientv3.WithCountOnly())
	if err != nil {
		logrus.Error(err)
		return false
	}
	return resp.Count > 0
}

func likeCount(statusID string) int64 {
	return countKeys(stateKey(fmt.Sprintf("/like/status/%s/", statusID)))
}

func LikeStatus(user *ActUser, statusID string) error {
	statusLikeKey := statusLikeKey(statusID, user.ID)
	userLikeKey := userLikeKey(statusID, user.ID)
	statueKey := stateKey(fmt.Sprintf("/status/%s", statusID))
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	s := GetStatus(statusID)
	if s == nil {
		return ErrStatusNotFound
	}
	ops := []clientv3.Op{
		clientv3.OpPut(statusLikeKey, string(b)),
		clientv3.OpPut(userLikeKey, statueKey),
	}
	ops = append(ops, newMessageOps(MsgOptions{
		from:     user,
		toUID:    s.User.ID,
		msgType:  MsgTypeLike,
		targetID: s.ID,
		message:  s.Overview(),
	})...)

	_, err = etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(statusLikeKey), ">", 0)).
		Then(clientv3.OpDelete(statusLikeKey), clientv3.OpDelete(userLikeKey)).
		Else(ops...).
		Commit()
	return err
}
