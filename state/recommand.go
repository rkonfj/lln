package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type CommentsRecommandMeta struct {
	Blacklist    []string `json:"blacklist"`
	RecommandID  string   `json:"recommandID"`
	comments     []*Status
	recommandKey string
	modRev       int64
}

func NewCommentsRecommandMeta(statusID string) (meta *CommentsRecommandMeta, err error) {
	recommandKey := stateKey(fmt.Sprintf("/meta/comments/recommended/%s", statusID))
	resp, err := etcdClient.KV.Get(context.Background(), recommandKey)
	if err != nil {
		return
	}
	if resp.Count > 0 {
		meta = &CommentsRecommandMeta{}
		err = json.Unmarshal(resp.Kvs[0].Value, &meta)
		if err != nil {
			return
		}
		meta.modRev = resp.Kvs[0].ModRevision
	} else {
		meta = &CommentsRecommandMeta{}
	}
	meta.recommandKey = recommandKey
	return
}

func NewCommentsRecommandMetaForRecommand(statusID string) (meta *CommentsRecommandMeta, err error) {
	meta, err = NewCommentsRecommandMeta(statusID)
	if err != nil {
		return
	}
	var createRev int64
	recommand := meta.Recommand()
	if recommand != nil {
		createRev = recommand.CreateRev
	}
	meta.comments, _ = StatusComments(statusID, &tools.PaginationOptions{
		After:  createRev,
		Size:   256,
		Ascend: true,
	})
	return
}

func (m *CommentsRecommandMeta) Recommand() *Status {
	if len(m.RecommandID) == 0 {
		return nil
	}
	return GetStatus(m.RecommandID)
}

func (m *CommentsRecommandMeta) RunRecommand(statusCreateRev int64) error {
	if len(m.comments) == 0 {
		return nil
	}
	var best *Status
	var bestScore float64
	for _, c := range m.comments {
		score := calcScore(c)
		if bestScore <= score {
			bestScore = score
			best = c
		}
	}

	r := m.Recommand()

	logrus.Debugf("thread %s comments best score is %4f", m.recommandKey, bestScore)

	if r != nil && calcScore(r) > bestScore {
		return nil
	}

	m.RecommandID = best.ID

	b, _ := json.Marshal(m)

	cmps := []clientv3.Cmp{}

	if m.modRev > 0 {
		cmps = append(cmps, clientv3.Compare(clientv3.ModRevision(m.recommandKey), "=", m.modRev))
	}

	resp, err := etcdClient.Txn(context.Background()).
		If(cmps...).
		Then(clientv3.OpPut(m.recommandKey, string(b)),
			clientv3.OpPut(lastStatusCreateRev, fmt.Sprintf("%d", statusCreateRev))).Commit()
	if err != nil {
		return nil
	}
	if !resp.Succeeded {
		return ErrTryAgainLater
	}
	return nil
}

func calcScore(c *Status) float64 {
	return float64((c.User.VerifiedCode+9)/10*2) + float64(c.Comments)*1.5 + float64(c.LikeCount)*1.2 + float64(c.Bookmarks)*1
}
