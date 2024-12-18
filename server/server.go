package main

import (
	"flag"
	"strconv"
	"sync"

	"github.com/gitferry/bamboo"
	"github.com/gitferry/bamboo/config"
	"github.com/gitferry/bamboo/crypto"
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/log"
	"github.com/gitferry/bamboo/replica"
)

// flag.string有三个参数，第一个参数是参数名称，第二个参数是默认值，第三个参数是帮助信息，它会在使用-help时候显示出来
var algorithm = flag.String("algorithm", "hotstuff", "BFT consensus algorithm")
var id = flag.String("id", "", "NodeID of the node")
var simulation = flag.Bool("sim", false, "simulation mode")

func initReplica(id identity.NodeID, isByz bool) {
	log.Infof("node %v starting...", id) //Infof是自定义函数
	if isByz {
		log.Infof("node %v is Byzantine", id)
	}

	r := replica.NewReplica(id, *algorithm, isByz)
	//*algorithm：共识算法的选择，它通过指针引用的方式获取之前定义的 algorithm 命令行参数的值。
	r.Start()
}

func main() {
	bamboo.Init()
	// the private and public keys are generated here
	errCrypto := crypto.SetKeys()
	if errCrypto != nil {
		log.Fatal("Could not generate keys:", errCrypto)
	}
	if *simulation { //处于模拟模式
		var wg sync.WaitGroup
		wg.Add(1)
		config.Simulation()
		for id := range config.GetConfig().Addrs {
			isByz := false
			if id.Node() <= config.GetConfig().ByzNo {
				isByz = true
			}
			go initReplica(id, isByz)
		}
		wg.Wait()
	} else {
		setupDebug()
		isByz := false
		i, _ := strconv.Atoi(*id)
		if i <= config.GetConfig().ByzNo {
			isByz = true
		}
		initReplica(identity.NodeID(*id), isByz)
	}
}
