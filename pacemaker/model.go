package pacemaker

import (
	"github.com/gitferry/bamboo/blockchain"
	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/types"
)

// TMO 代表 "Timeout Message"，即超时消息
type TMO struct {
	View   types.View
	NodeID identity.NodeID
	HighQC *blockchain.QC
}

// TC 代表 "Timeout Certificate"，即超时证明
type TC struct {
	types.View
	crypto.AggSig
	crypto.Signature
}

func NewTC(view types.View, requesters map[identity.NodeID]*TMO) *TC {
	// TODO: add crypto
	return &TC{View: view}
}
