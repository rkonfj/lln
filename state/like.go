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
	statusLikeSetKey := statusLikeKey(statusID, user.ID)
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	s := GetStatus(statusID)
	if s == nil {
		return ErrStatusNotFound
	}
	ops := []clientv3.Op{clientv3.OpPut(statusLikeSetKey, string(b))}
	ops = append(ops, newMessageOps(MsgOptions{
		from:     user,
		toUID:    s.User.ID,
		msgType:  MsgTypeLike,
		targetID: s.ID,
		message:  s.Overview(),
	})...)

	_, err = etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(statusLikeSetKey), ">", 0)).
		Then(clientv3.OpDelete(statusLikeSetKey)).
		Else(ops...).
		Commit()
	return err
}
