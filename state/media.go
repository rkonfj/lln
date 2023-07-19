package state

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func SaveMedia(user *ActUser, objectPath string) error {
	_, err := etcdClient.KV.Put(context.Background(),
		stateKey(fmt.Sprintf("/media/%s%s", user.ID, objectPath)), "")
	if err != nil {
		return err
	}
	return nil
}

func TodayMediaCountByUser(user *ActUser) int64 {
	timePrefix := time.Now().Format("20060102")
	resp, err := etcdClient.KV.Get(context.Background(),
		stateKey(fmt.Sprintf("/media/%s/%s", user.ID, timePrefix)),
		clientv3.WithCountOnly(), clientv3.WithPrefix())
	if err != nil {
		logrus.Error("query media count etcd error: ", err)
		return 0
	}
	return resp.Count
}
