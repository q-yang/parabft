package parabft

import (
	"fmt"
	"sync"

	"github.com/gitferry/bamboo/blockchain"
	"github.com/gitferry/bamboo/config"
	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/election"
	"github.com/gitferry/bamboo/log"
	"github.com/gitferry/bamboo/message"
	"github.com/gitferry/bamboo/node"
	"github.com/gitferry/bamboo/pacemaker"
	"github.com/gitferry/bamboo/types"
)

const FORK = "fork"

type Parabft struct {
	node.Node
	election.Election
	pm              *pacemaker.Pacemaker //Pacemaker 用于同步各个节点的视图和时间，确保节点在同一个时间上进行共识。
	lastVotedView   types.View           //储存节点上次投票的视图
	preferredView   types.View
	highQC          *blockchain.QC
	bc              *blockchain.BlockChain //表示节点维护的区块链。这个属性包含了节点所知道的所有区块信息，包括已提交的和未提交的区块。
	committedBlocks chan *blockchain.Block
	forkedBlocks    chan *blockchain.Block
	bufferedQCs     map[crypto.Identifier]*blockchain.QC //bufferedQCs 属性：用于缓存待处理的区块证明（QC）。
	bufferedBlocks  map[types.View]*blockchain.Block
	mu              sync.Mutex
}

// GetChainStatus implements replica.Safety.
// func (*HotStuffzg) GetChainStatus() string {
// 	panic("unimplemented")
// }

// // ProcessLocalTmo implements replica.Safety.
// func (*HotStuffzg) ProcessLocalTmo(view types.View) {
// 	panic("unimplemented")
// }

// // ProcessRemoteTmo implements replica.Safety.
// func (*HotStuffzg) ProcessRemoteTmo(tmo *pacemaker.TMO) {
// 	panic("unimplemented")
// }

func NewParabft(
	node node.Node,
	pm *pacemaker.Pacemaker,
	elec election.Election,
	committedBlocks chan *blockchain.Block,
	forkedBlocks chan *blockchain.Block) *Parabft {
	hs := new(Parabft)
	hs.Node = node
	hs.Election = elec
	hs.pm = pm
	hs.bc = blockchain.NewBlockchain(config.GetConfig().N())
	hs.bufferedBlocks = make(map[types.View]*blockchain.Block)
	hs.bufferedQCs = make(map[crypto.Identifier]*blockchain.QC)
	hs.highQC = &blockchain.QC{View: 0}
	hs.committedBlocks = committedBlocks
	hs.forkedBlocks = forkedBlocks
	return hs
}

// 这个是在replica.go里面调用的
func (hs *Parabft) ProcessBlock(block *blockchain.Block) error {
	log.Debugf("[%v] is processing block from %v, view: %v, id: %x", hs.ID(), block.Proposer.Node(), block.View, block.ID)

	hs.bc.AddBlock(block)
	// process buffered QC
	qc, ok := hs.bufferedQCs[block.ID]
	if ok {
		hs.processCertificate(qc)
		delete(hs.bufferedQCs, block.ID)
	}
	vote := blockchain.MakeVote(block.View, hs.ID(), block.ID)
	//MakeVote生成投票，投票是包含视图号的，可以改为处理投票时候与视图号无关
	// vote is sent to the next leader
	//voteAggregator := hs.FindLeaderFor(block.View + 1)
	//非流水线
	// voteAggregator := block.Proposer
	voteAggregator := hs.FindLeaderFor(block.View + 1)
	if voteAggregator == hs.ID() {
		log.Debugf("[%v] vote is sent to itself, id: %x", hs.ID(), vote.BlockID)
		hs.ProcessVote(vote)
	} else {
		log.Debugf("[%v] vote is sent to %v, id: %x", hs.ID(), voteAggregator, vote.BlockID)
		hs.Send(voteAggregator, vote)
	}
	b, ok := hs.bufferedBlocks[block.View]
	if ok {
		_ = hs.ProcessBlock(b)
		delete(hs.bufferedBlocks, block.View)
	}

	return nil
}

// 只有投票的搜集者才处理投票
func (hs *Parabft) ProcessVote(vote *blockchain.Vote) {
	log.Debugf("[%v] is processing the vote, block id: %x", hs.ID(), vote.BlockID)

	//？？？没看懂这部分干嘛的
	//因为只有应该搜集本区块投票的节点才执行ProcessVote，判断一下投票是否够了
	//AddVote用来判断投票是否达到了大多数
	isBuilt, qc := hs.bc.AddVote(vote)
	// _, qc := hs.bc.AddVote(vote)
	//投票的时候需要区块ID，可以关注一些区块ID是怎么生成的
	if !isBuilt {
		log.Debugf("[%v] not sufficient votes to build a QC, block id: %x", hs.ID(), vote.BlockID)
		return
	}
	//投票数达到了大多数就可以处理QC了
	qc.Leader = hs.ID()

	hs.processCertificate(qc) //搜集的投票数超过2/3才调用processCertificate，提交区块就在这里面

}
func (hs *Parabft) ProcessRemoteTmo(tmo *pacemaker.TMO) {
	log.Debugf("[%v] is processing tmo from %v", hs.ID(), tmo.NodeID)
	hs.processCertificate(tmo.HighQC)
	isBuilt, tc := hs.pm.ProcessRemoteTmo(tmo)
	if !isBuilt {
		return
	}
	log.Debugf("[%v] a tc is built for view %v", hs.ID(), tc.View)
	hs.processTC(tc)
}

func (hs *Parabft) ProcessLocalTmo(view types.View) {
	hs.pm.AdvanceView(view)
	tmo := &pacemaker.TMO{
		View:   view + 1,
		NodeID: hs.ID(),
		HighQC: hs.GetHighQC(),
	}
	hs.Broadcast(tmo)
	hs.ProcessRemoteTmo(tmo)
}
func (hs *Parabft) MakeProposal(view types.View, payload []*message.Transaction) *blockchain.Block {
	// qc := hs.forkChoice()
	qc := hs.GetHighQC()
	block := blockchain.MakeBlock(view, qc, qc.BlockID, payload, hs.ID())
	//可以尝试一下让前哈希等于自己的ID
	return block
}
func (hs *Parabft) processTC(tc *pacemaker.TC) {
	if tc.View < hs.pm.GetCurView() {
		return
	}
	hs.pm.AdvanceView(tc.View)
}
func (hs *Parabft) GetHighQC() *blockchain.QC {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	return hs.highQC
}

func (hs *Parabft) updateHighQC(qc *blockchain.QC) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if qc.View > hs.highQC.View {
		hs.highQC = qc
	}
}

func (hs *Parabft) processCertificate(qc *blockchain.QC) {
	log.Debugf("[%v] is processing a QC, block id: %x", hs.ID(), qc.BlockID)
	if qc.View < hs.pm.GetCurView() {
		return
	}
	if qc.Leader != hs.ID() {
		quorumIsVerified, _ := crypto.VerifyQuorumSignature(qc.AggSig, qc.BlockID, qc.Signers)
		if quorumIsVerified == false {
			log.Warningf("[%v] received a quorum with invalid signatures", hs.ID())
			return
		}
	}
	hs.pm.AdvanceView(qc.View)
	hs.updateHighQC(qc)
	if qc.View < 3 {
		return
	}
	ok, block, _ := hs.commitRule(qc)
	if !ok {
		return
	}
	// forked blocks are found when pruning
	// committedBlocks, forkedBlocks, err := hs.bc.CommitBlock(block.ID, hs.pm.GetCurView())
	committedBlocks, _, err := hs.bc.CommitBlock(block.ID, hs.pm.GetCurView())
	//提交这里跟视图号没什么太大的关系
	if err != nil {
		log.Errorf("[%v] cannot commit blocks, %w", hs.ID(), err)
		return
	}
	for _, cBlock := range committedBlocks {
		hs.committedBlocks <- cBlock
	}
	// for _, fBlock := range forkedBlocks {
	// 	hs.forkedBlocks <- fBlock
	// }
}
func (hs *Parabft) GetChainStatus() string {
	chainGrowthRate := hs.bc.GetChainGrowth()
	blockIntervals := hs.bc.GetBlockIntervals()
	return fmt.Sprintf("[%v] The current view is: %v, chain growth rate is: %v, ave block interval is: %v", hs.ID(), hs.pm.GetCurView(), chainGrowthRate, blockIntervals)
}
func (hs *Parabft) commitRule(qc *blockchain.QC) (bool, *blockchain.Block, error) {
	parentBlock, err := hs.bc.GetParentBlock(qc.BlockID)
	if err != nil {
		return false, nil, fmt.Errorf("cannot commit any block: %w", err)
	}
	grandParentBlock, err := hs.bc.GetParentBlock(parentBlock.ID)
	if err != nil {
		return false, nil, fmt.Errorf("cannot commit any block: %w", err)
	}
	if ((grandParentBlock.View + 1) == parentBlock.View) && ((parentBlock.View + 1) == qc.View) {
		return true, grandParentBlock, nil
	}
	return false, nil, nil
	// bolck, err := hs.bc.GetBlockByID(qc.BlockID)
	// if err != nil {
	// 	return false, nil, fmt.Errorf("cannot commit any block: %w", err)
	// }
	// return true, bolck, nil
}
