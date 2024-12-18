package election

import (
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/types"
)

// Election 的接口，它规定了一些用于选举（Election）操作的方法和行为。选举通常在分布式系统中用于选择领导者或主节点
type Election interface {
	IsLeader(id identity.NodeID, view types.View) bool
	FindLeaderFor(view types.View) identity.NodeID
	//FindLeaderFor1(view types.View) identity.NodeID
}
