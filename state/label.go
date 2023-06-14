package state

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Label struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

func GetLabels(prefix string) (labels []*Label) {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf("/label/%s", prefix)),
		clientv3.WithSort(clientv3.SortByVersion, clientv3.SortDescend),
		clientv3.WithPrefix())
	if err != nil {
		logrus.Debug(err)
		return
	}
	for _, kv := range resp.Kvs {
		labels = append(labels, &Label{Value: string(kv.Value), Count: kv.Version})
	}
	return
}
