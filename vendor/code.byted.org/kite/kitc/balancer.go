package kitc

import (
	"math/rand"
	"strconv"

	"code.byted.org/kite/kitutil"
	"code.byted.org/kite/kitware"
)

var connBalanceRetrier = newConnBalanceRetrier()

const (
	_MAX_CONN_RETRY_TIMES    = 3
	_MIN_CONN_SAMPLES        = 200
	_DEFAULT_CONN_RETRY_RATE = .1
)

// weightBalancer select instance according instance's weight
type weightBalancer struct {
	ins        []kitutil.Instance
	weightList []int
	i          int
	sum        int
}

func newWeightBalancer(ins []kitutil.Instance) *weightBalancer {
	weightList := make([]int, len(ins))
	sum := 0
	for i, in := range ins {
		if weight, ok := in.Tags()["weight"]; ok {
			val, err := strconv.ParseInt(weight, 10, 64)
			if err != nil {
				val = 100
			}
			weightList[i] = int(val)
		} else {
			weightList[i] = 100
		}
		sum += weightList[i]
	}
	return &weightBalancer{
		ins:        ins,
		weightList: weightList,
		sum:        sum,
	}
}

// SelectOne return an instance according instance's weight
func (wb *weightBalancer) SelectOne() kitutil.Instance {
	if wb.i > len(wb.ins)-1 {
		return nil
	}
	if wb.i < len(wb.ins)-1 {
		if wb.sum == 0 {
			// all remaining instances' weight are zero now
			wb.i = len(wb.ins)
			return nil
		}
		rd := rand.Intn(wb.sum)
		j := wb.i
		for rd >= 0 {
			rd -= wb.weightList[j]
			if rd >= 0 {
				j++
			}
		}
		wb.ins[wb.i], wb.ins[j] = wb.ins[j], wb.ins[wb.i]
		wb.weightList[wb.i], wb.weightList[j] = wb.weightList[j], wb.weightList[wb.i]
		wb.sum = wb.sum - wb.weightList[wb.i]
	}
	wb.i++
	return wb.ins[wb.i-1]
}

// balanceRetrier used to do conn LB and retry
type balanceRetrier struct{}

func newConnBalanceRetrier() *balanceRetrier {
	return &balanceRetrier{}
}

// CreateBalancer .
func (br *balanceRetrier) CreateBalancer(ins []kitutil.Instance) (kitware.Balancer, error) {
	return newWeightBalancer(ins), nil
}
