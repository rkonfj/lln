package state

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client

func InitState(endpoints []string) (err error) {
	etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	return
}

func stateKey(key string) string {
	if strings.HasPrefix(key, "/") {
		return fmt.Sprintf("/lln%s", key)
	}
	return fmt.Sprintf("/lln/%s", key)
}

func getPointerValue(key string) (*clientv3.GetResponse, error) {
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) != 1 {
		return nil, errors.New("not found")
	}
	return etcdClient.KV.Get(context.Background(), string(resp.Kvs[0].Value))
}
