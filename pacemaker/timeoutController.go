package pacemaker

import (
	"sync"

	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/types"
)

type TimeoutController struct {
	n        int                                     // the size of the network
	timeouts map[types.View]map[identity.NodeID]*TMO // keeps track of timeout msgs
	mu       sync.Mutex
}

func NewTimeoutController(n int) *TimeoutController {
	tcl := new(TimeoutController)
	tcl.n = n
	tcl.timeouts = make(map[types.View]map[identity.NodeID]*TMO)
	return tcl
}

func (tcl *TimeoutController) AddTmo(tmo *TMO) (bool, *TC) {
	tcl.mu.Lock()
	defer tcl.mu.Unlock()
	if tcl.superMajority(tmo.View) {
		return false, nil
		//return true, NewTC(tmo.View, tcl.timeouts[tmo.View])
	}
	//果 tcl.timeouts 映射中已经存在了与 tmo.View 相关联的条目，那么 exist 将被设置为 true。
	//如果 tcl.timeouts 映射中没有与 tmo.View 相关联的条目，那么 exist 将被设置为 false
	_, exist := tcl.timeouts[tmo.View]
	if !exist {
		//	first time of receiving the timeout for this view
		tcl.timeouts[tmo.View] = make(map[identity.NodeID]*TMO)
	}
	tcl.timeouts[tmo.View][tmo.NodeID] = tmo
	if tcl.superMajority(tmo.View) {
		return true, NewTC(tmo.View, tcl.timeouts[tmo.View])
	}

	return false, nil
}

func (tcl *TimeoutController) superMajority(view types.View) bool {
	return tcl.total(view) > tcl.n*2/3
}

func (tcl *TimeoutController) total(view types.View) int {
	return len(tcl.timeouts[view])
}
