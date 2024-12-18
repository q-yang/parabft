package node

import (
	"net/http"
	"reflect"
	"sync"

	"github.com/gitferry/bamboo/config"
	"github.com/gitferry/bamboo/identity"
	"github.com/gitferry/bamboo/log"
	"github.com/gitferry/bamboo/message"
	"github.com/gitferry/bamboo/socket"
)

// Node is the primary access point for every replica
// it includes networking, state machine and RESTful API server
type Node interface {
	socket.Socket
	//Database
	ID() identity.NodeID
	Run()
	Retry(r message.Transaction)                       //这是一个方法，用于重试特定的交易。通常，节点可能需要在某些情况下重新尝试执行交易。
	Forward(id identity.NodeID, r message.Transaction) //实现交易的传递
	Register(m interface{}, f interface{})
	IsByz() bool
}

// node implements Node interface
type node struct {
	id identity.NodeID

	socket.Socket
	//Database
	MessageChan chan interface{}
	TxChan      chan interface{}
	handles     map[string]reflect.Value
	server      *http.Server
	isByz       bool
	totalTxn    int

	sync.RWMutex
	forwards map[string]*message.Transaction
}

// NewNode creates a new Node object from configuration
func NewNode(id identity.NodeID, isByz bool) Node {
	return &node{
		id:     id,
		isByz:  isByz,
		Socket: socket.NewSocket(id, config.Configuration.Addrs),
		//NewSocket 函数：用于创建 Socket 接口的新实例。它需要节点的标识和其他节点的地址映射，然后创建一个 socket 结构体，并为当前节点建立传输连接。
		//Database:    NewDatabase(),
		MessageChan: make(chan interface{}, config.Configuration.ChanBufferSize), //onfig.Configuration.ChanBufferSize 是配置文件中定义的缓冲大小。
		TxChan:      make(chan interface{}, config.Configuration.ChanBufferSize),
		handles:     make(map[string]reflect.Value),
		forwards:    make(map[string]*message.Transaction),
	}
}

// 返回节点标识
func (n *node) ID() identity.NodeID {
	return n.id
}

func (n *node) IsByz() bool {
	return n.isByz
}

func (n *node) Retry(r message.Transaction) {
	log.Debugf("node %v retry reqeust %v", n.id, r)
	n.MessageChan <- r
}

// Register a handle function for each message type
func (n *node) Register(m interface{}, f interface{}) {
	t := reflect.TypeOf(m)   //用于获取消息类型 m 的反射对象，即获取消息类型的元数据信息。例如类型名称、字段、方法等。
	fn := reflect.ValueOf(f) // 包含了有关函数值的元数据信息，例如函数的名称、输入参数、输出参数等。这也是只读操作，用于获取函数值的信息而不修改它。

	if fn.Kind() != reflect.Func {
		panic("handle function is not func")
	}
	/*
	   fn.Kind() != reflect.Func 这行代码的意思是：如果反射对象 fn 不代表一个函数类型，也就是 fn 的底层类型不是函数类型，
	   那么就会抛出一个 panic，意味着处理函数不是一个函数。
	   这种检查通常用于确保在注册函数处理程序时传递了有效的函数。
	   在某些情况下，你可能期望传递一个函数处理程序，
	   但如果传递了其他类型的值，程序可能无法正确工作。因此，这是一种防御性编程的方式，以确保代码的健壮性。
	*/
	if fn.Type().In(0) != t {
		panic("func type is not t")
	}

	if fn.Kind() != reflect.Func || fn.Type().NumIn() != 1 || fn.Type().In(0) != t {
		panic("register handle function error")
	}
	n.handles[t.String()] = fn
}

// Run start and run the node运行节点
func (n *node) Run() {
	log.Infof("node %v start running", n.id)
	if len(n.handles) > 0 {
		go n.handle() //它会不断从消息通道中接收消息，并根据消息类型调用相应的处理函数。
		go n.recv()   //方法会从节点的网络连接中接收消息，并将其放入消息通道中，以便稍后被处理。
		go n.txn()    //这行代码启动另一个 Goroutine，用于处理交易。节点会从交易通道中接收交易，并将其传递给注册的处理函数。
	}
	n.http()
}

// 这个函数是用于处理从 n.TxChan 中接收的消息，根据消息类型查找并调用相应的处理函数。
func (n *node) txn() {
	for {
		tx := <-n.TxChan
		v := reflect.ValueOf(tx)
		name := v.Type().String()
		f, exists := n.handles[name]
		if !exists {
			log.Fatalf("no registered handle function for message type %v", name)
		}
		f.Call([]reflect.Value{v})
	}
}

// recv receives messages from socket and pass to message channel
// 用于处理接收到的消息
// 这段代码是 node 结构中负责接收消息并进行初步处理的方法。它根据消息类型分别处理事务消息和回复消息，同时允许节点进行沉默攻击。
func (n *node) recv() {
	for {
		m := n.Recv() //.Recv() 通常是 socket 包中的 Socket 接口的实现之一，用于从节点的网络连接中接收消息。
		if n.isByz && config.GetConfig().Strategy == "silence" {
			// perform silence attack
			continue
		}
		switch m := m.(type) {
		case message.Transaction:
			m.C = make(chan message.TransactionReply, 1)
			n.TxChan <- m
			continue

		case message.TransactionReply:
			n.RLock()
			r := n.forwards[m.Command.String()]
			log.Debugf("node %v received reply %v", n.id, m)
			n.RUnlock()
			r.Reply(m)
			continue
		}
		n.MessageChan <- m
	}
}

// handle receives messages from message channel and calls handle function using refection
// 它的主要作用是接收来自 MessageChan 的消息，并使用反射来查找并调用相应的处理函数。
func (n *node) handle() {
	for {
		msg := <-n.MessageChan
		v := reflect.ValueOf(msg)
		name := v.Type().String()
		f, exists := n.handles[name]
		if !exists {
			log.Fatalf("no registered handle function for message type %v", name)
		}
		f.Call([]reflect.Value{v})
	}
}

/*
func (n *node) Forward(id NodeID, m Transaction) {
	key := m.Command.Key
	url := config.HTTPAddrs[id] + "/" + strconv.Itoa(int(key))

	log.Debugf("Node %v forwarding %v to %s", n.NodeID(), m, id)

	method := http.MethodGet
	var body io.Reader
	if !m.Command.IsRead() {
		method = http.MethodPut
		body = bytes.NewBuffer(m.Command.Value)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Error(err)
		return
	}
	req.Header.Set(HTTPClientID, string(n.id))
	req.Header.Set(HTTPCommandID, strconv.Itoa(m.Command.CommandID))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
		m.TransactionReply(TransactionReply{
			Command: m.Command,
			Err:     err,
		})
		return
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error(err)
		}
		m.TransactionReply(TransactionReply{
			Command: m.Command,
			Value:   Value(b),
		})
	} else {
		m.TransactionReply(TransactionReply{
			Command: m.Command,
			Err:     errors.New(res.Status),
		})
	}
}
*/
//总的来说，这段代码表示了节点接收到一条消息后，将该消息标记为已发送，并将其转发给另一个节点，同时记录日志以跟踪转发操作。
//这种机制通常用于在分布式系统中传递消息或请求，以实现分布式计算或协同工作。
func (n *node) Forward(id identity.NodeID, m message.Transaction) {
	log.Debugf("Node %v forwarding %v to %s", n.ID(), m, id)
	m.NodeID = n.id
	n.Lock()
	n.forwards[m.Command.String()] = &m
	n.Unlock()
	n.Send(id, m)
}

/*
这段代码描述了节点对象的实现，包括消息处理、网络通信和节点的运行。
节点可以注册消息处理函数，并通过不同的 Goroutines 处理来自网络的消息，以及通过 HTTP 服务器提供服务
*/
