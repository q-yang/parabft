package blockchain

import (
	"fmt"

	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/log"
	"github.com/gitferry/bamboo/types"
)

type Vote struct {
	types.View
	Voter   identity.NodeID
	BlockID crypto.Identifier
	crypto.Signature
}

// 确认证明（Quorum Certificate）
type QC struct {
	Leader  identity.NodeID
	View    types.View
	BlockID crypto.Identifier
	Signers []identity.NodeID
	crypto.AggSig
	crypto.Signature
}

type Quorum struct {
	total int
	votes map[crypto.Identifier]map[identity.NodeID]*Vote
	//crypto.Identifier 通常表示某个数据块（如区块）的唯一标识，
	//而 identity.NodeID 表示参与者（节点）的唯一标识。
}

// MakeVote 函数：用于创建一个投票对象。
// 它接受视图、投票者和区块ID，然后使用投票者的私钥对区块ID进行签名，生成投票对象。
func MakeVote(view types.View, voter identity.NodeID, id crypto.Identifier) *Vote {
	// TODO: uncomment the following
	//这函数 IDToByte 的目的是将 crypto.Identifier 类型的标识符
	//（通常是一个固定长度的字节序列）转换为普通的字节切片（[]byte）。
	sig, err := crypto.PrivSign(crypto.IDToByte(id), voter, nil)
	if err != nil {
		log.Fatalf("[%v] has an error when signing a vote", voter)
		return nil
	}
	return &Vote{
		View:      view,
		Voter:     voter,
		BlockID:   id,
		Signature: sig,
	}
}

// NewQuorum 函数：用于创建一个法定人数对象。
func NewQuorum(total int) *Quorum {
	return &Quorum{
		total: total,
		votes: make(map[crypto.Identifier]map[identity.NodeID]*Vote),
	}
}

// Add adds id to quorum ack records
func (q *Quorum) Add(vote *Vote) (bool, *QC) {
	if q.superMajority(vote.BlockID) {
		//if q.SuperMajority(vote.BlockID) {
		return false, nil
	}
	_, exist := q.votes[vote.BlockID]
	if !exist {
		//	first time of receiving the vote for this block
		q.votes[vote.BlockID] = make(map[identity.NodeID]*Vote)
	}
	q.votes[vote.BlockID][vote.Voter] = vote
	if q.superMajority(vote.BlockID) {
		//if q.SuperMajority(vote.BlockID) {
		aggSig, signers, err := q.getSigs(vote.BlockID)
		if err != nil {
			log.Warningf("cannot generate a valid qc, view: %v, block id: %x: %w", vote.View, vote.BlockID, err)
		}
		qc := &QC{
			View:    vote.View,
			BlockID: vote.BlockID,
			AggSig:  aggSig,
			Signers: signers,
		}
		return true, qc
	}
	return false, nil
}

// Super majority quorum satisfied
func (q *Quorum) superMajority(blockID crypto.Identifier) bool {
	//func (q *Quorum) SuperMajority(blockID crypto.Identifier) bool {
	return q.size(blockID) > q.total*2/3
}

// Size returns ack size for the block
func (q *Quorum) size(blockID crypto.Identifier) int {
	return len(q.votes[blockID])
}

// 这个函数的主要作用是为了准备生成确认证明（QC）所需的聚合签名信息和签名者信息。
// 确认证明用于证明一组节点已经就某个区块达成一致，需要包含该区块的所有签名信息和签名者信息。
func (q *Quorum) getSigs(blockID crypto.Identifier) (crypto.AggSig, []identity.NodeID, error) {
	var sigs crypto.AggSig
	var signers []identity.NodeID
	_, exists := q.votes[blockID]
	if !exists {
		return nil, nil, fmt.Errorf("sigs does not exist, id: %x", blockID)
	}
	for _, vote := range q.votes[blockID] {
		sigs = append(sigs, vote.Signature)
		signers = append(signers, vote.Voter)
	}

	return sigs, signers, nil
}
