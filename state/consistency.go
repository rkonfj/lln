package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	UserChanged chan *User = make(chan *User, 128)
)

func init() {
	go keepStatusUserConsistentLoop()
}

func keepStatusUserConsistentLoop() {
	for e := range UserChanged {
		lastKey := ""
		for {
			ss := e.ListStatus(lastKey, 20)
			if ss == nil {
				break
			}
			for i, s := range ss {
				if i == len(ss)-1 {
					lastKey = s.ID
				}
				if s.User.Name != e.Name || s.User.UniqueName != e.UniqueName {
					key := stateKey(fmt.Sprintf("/status/%s", s.ID))
					s.User.Name = e.Name
					s.User.UniqueName = e.UniqueName
					b, err := json.Marshal(s)
					if err != nil {
						logrus.Debug(err)
						continue
					}
					_, err = etcdClient.Put(context.Background(), key, string(b))
					if err != nil {
						logrus.Error(err)
					}
				}
			}
		}
	}
}
