package state

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func BookmarkStatus(user *ActUser, statusID string) error {
	bookmarkKey := stateKey(fmt.Sprintf("/bookmark/%s/%s", user.ID, statusID))
	s := GetStatus(statusID)
	if s == nil {
		return ErrStatusNotFound
	}
	ops := []clientv3.Op{clientv3.OpPut(bookmarkKey, stateKey(fmt.Sprintf("/status/%s", statusID)))}
	ops = append(ops, newMessageOps(MsgOptions{
		from:     user,
		toUID:    s.User.ID,
		msgType:  MsgTypeBookmark,
		targetID: s.ID,
		message:  s.Overview(),
	})...)

	_, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(bookmarkKey), ">", 0)).
		Then(clientv3.OpDelete(bookmarkKey)).
		Else(ops...).
		Commit()
	return err
}

func ListBookmarks(user *ActUser, after string, size int64) []*Status {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/bookmark/%s", user.ID)), after, size)
}
