package state

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func BookmarkStatus(user *ActUser, statusID string) error {
	bookmarkKey := stateKey(fmt.Sprintf("/bookmark/%s/%s", user.ID, statusID))
	_, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(bookmarkKey), ">", 0)).
		Then(clientv3.OpDelete(bookmarkKey)).
		Else(clientv3.OpPut(bookmarkKey, stateKey(fmt.Sprintf("/status/%s", statusID)))).
		Commit()
	return err
}

func ListBookmarks(user *ActUser, after string, size int64) []*Status {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/bookmark/%s", user.ID)), after, size)
}
