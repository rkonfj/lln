package state

import (
	"context"
	"encoding/json"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func BookmarkStatus(user *ActUser, statusID string) error {
	s := GetStatus(statusID)
	if s == nil {
		return ErrStatusNotFound
	}
	b, _ := json.Marshal(user)
	bookmarkKey := stateKey(fmt.Sprintf("/bookmark/%s/%s", user.ID, statusID))
	bookmarkStatusKey := stateKey(fmt.Sprintf("/bookmark/status/%s/%s", statusID, user.ID))

	ops := []clientv3.Op{
		clientv3.OpPut(bookmarkKey, stateKey(fmt.Sprintf("/status/%s", statusID))),
		clientv3.OpPut(bookmarkStatusKey, string(b)),
	}
	ops = append(ops, newMessageOps(MsgOptions{
		from:     user,
		toUID:    s.User.ID,
		msgType:  MsgTypeBookmark,
		targetID: s.ID,
		message:  s.Overview(),
	})...)

	_, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(bookmarkKey), ">", 0)).
		Then(clientv3.OpDelete(bookmarkKey), clientv3.OpDelete(bookmarkStatusKey)).
		Else(ops...).
		Commit()
	return err
}

func ListBookmarks(user *ActUser, after string, size int64) []*Status {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/bookmark/%s", user.ID)), after, size)
}
