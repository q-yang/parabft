package blockchain

import (
	"time"

	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/message"
	"github.com/gitferry/bamboo/types"
)

// 区块的结构
type Block struct {
	types.View
	QC        *QC
	Proposer  identity.NodeID
	Timestamp time.Time
	Payload   []*message.Transaction
	PrevID    crypto.Identifier
	Sig       crypto.Signature
	ID        crypto.Identifier
	Ts        time.Duration
}

type rawBlock struct {
	types.View
	QC       *QC
	Proposer identity.NodeID
	Payload  []string
	PrevID   crypto.Identifier
	Sig      crypto.Signature
	ID       crypto.Identifier
}

// MakeBlock creates an unsigned block
func MakeBlock(view types.View, qc *QC, prevID crypto.Identifier, payload []*message.Transaction, proposer identity.NodeID) *Block {
	b := new(Block)
	b.View = view
	b.Proposer = proposer
	b.QC = qc
	b.Payload = payload
	b.PrevID = prevID
	b.makeID(proposer)

	return b
}

func (b *Block) makeID(nodeID identity.NodeID) {
	raw := &rawBlock{
		View:     b.View,
		QC:       b.QC,
		Proposer: b.Proposer,
		PrevID:   b.PrevID,
	}
	var payloadIDs []string
	/*
		生成 Payload ID 列表:
		在 Block 结构体中，有一个字段叫做 Payload，表示区块中包含的一组交易。
		此处，代码遍历这些交易，提取每个交易的唯一标识符（ID），并将这些 ID 存储在 payloadIDs 列表中。
	*/
	for _, txn := range b.Payload {
		payloadIDs = append(payloadIDs, txn.ID)
	}
	raw.Payload = payloadIDs
	b.ID = crypto.MakeID(raw)
	// TODO: uncomment the following
	b.Sig, _ = crypto.PrivSign(crypto.IDToByte(b.ID), nodeID, nil)
}
