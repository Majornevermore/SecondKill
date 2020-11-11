package loadbalance

import (
	"SecondKill/pkg/common"
	"errors"
	"math/rand"
)

type Balance interface {
	SelectBalance(serives []*common.ServiceInstance) (*common.ServiceInstance, error)
}

type RandomBalance struct{}

func (RandomBalance) SelectBalance(serives []*common.ServiceInstance) (*common.ServiceInstance, error) {
	if serives == nil || len(serives) == 0 {
		return nil, errors.New("service instance are not exist")
	}
	return serives[rand.Intn(len(serives))], nil
}

type WeightRoundRobinLoadBalance struct{}

func (WeightRoundRobinLoadBalance) SelectBalance(serives []*common.ServiceInstance) (best *common.ServiceInstance, err error) {
	if serives == nil || len(serives) == 0 {
		return nil, errors.New("service instance are not exist")
	}
	total := 0
	for i := 0; i < len(serives); i++ {
		w := serives[i]
		if w == nil {
			continue
		}
		total += w.Weight
		w.CurWeight += w.Weight
		if best == nil || w.CurWeight > best.CurWeight {
			best = w
		}
	}
	if best == nil {
		return nil, errors.New("service instance are not exist")
	}
	best.CurWeight -= total
	return best, nil
}

type SuffleBalance struct{}

func (SuffleBalance) SelectBalance(serives []*common.ServiceInstance) (*common.ServiceInstance, error) {
	if serives == nil || len(serives) < 1 {
		return nil, errors.New("service instance are not exist")
	}
	//for i:= len(serives); i>0; i-- {
	//	lastIdex := i -1
	//	idx := rand.Intn(i)
	//	serives[lastIdex], serives[idx] = serives[idx], serives[lastIdex]
	//}
	b := rand.Perm(len(serives))
	return serives[b[0]], nil
}
