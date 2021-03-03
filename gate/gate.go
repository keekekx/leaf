package gate

import (
	"container/list"
	"github.com/keekekx/leaf/chanrpc"
	"github.com/keekekx/leaf/log"
	"github.com/keekekx/leaf/network"
	"github.com/keekekx/leaf/util"
	"net"
	"reflect"
	"time"
)

type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32
	Processor       network.Processor
	AgentChanRPC    *chanrpc.Server

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	TCPAddr      string
	LenMsgLen    int
	LittleEndian bool

	CreateErrorResp func(e *util.ErrorInfo) interface{}
}

func (gate *Gate) Run(closeSig chan bool) {
	var wsServer *network.WSServer
	if gate.WSAddr != "" {
		wsServer = new(network.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.PendingWriteNum = gate.PendingWriteNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
		wsServer.NewAgent = func(conn *network.WSConn) network.Agent {
			a := &agent{conn: conn, gate: gate, respBuff: list.New()}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		}
	}

	var tcpServer *network.TCPServer
	if gate.TCPAddr != "" {
		tcpServer = new(network.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.NewAgent = func(conn *network.TCPConn) network.Agent {
			a := &agent{conn: conn, gate: gate, respBuff: list.New()}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		}
	}

	if wsServer != nil {
		wsServer.Start()
	}
	if tcpServer != nil {
		tcpServer.Start()
	}
	<-closeSig
	if wsServer != nil {
		wsServer.Close()
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
}

func (gate *Gate) OnDestroy() {}

type respIns struct {
	ctx  uint32
	data interface{}
}

type agent struct {
	conn     network.Conn
	gate     *Gate
	userData interface{}
	respBuff *list.List
}

func (a *agent) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("gate has some error!")
		}
	}()

	for {
		//历史消息缓冲，同ctx请求时，认为已经处理过，回复历史记录
		if a.respBuff.Len() > 3 {
			a.respBuff.Remove(a.respBuff.Back())
		}
		data, err := a.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}

		if a.gate.Processor != nil {
			ctx, msg, err := a.gate.Processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}

			useBuff := false
			for i := a.respBuff.Front(); i != nil; i = i.Next() {
				r := i.Value.(*respIns)
				if r.ctx == ctx {
					a.RespMsg(ctx, r.data)
					useBuff = true
					break
				}
			}

			if useBuff {
				continue
			}

			resp, err := a.gate.Processor.Route(msg, a)

			if err != nil && a.gate.CreateErrorResp != nil {
				if e, ok := err.(*util.ErrorInfo); ok {
					r := a.gate.CreateErrorResp(e)
					a.RespMsg(ctx, r)
					if ctx > 0 {
						a.respBuff.PushFront(&respIns{
							ctx:  ctx,
							data: r,
						})
					}
					if e.Kick {
						break
					}
				}
				log.Debug("message error: %v", err)
			} else if resp != nil {
				a.RespMsg(ctx, resp)
				if ctx > 0 {
					a.respBuff.PushFront(&respIns{
						ctx:  ctx,
						data: resp,
					})
				}
			}
		}
	}
}

func (a *agent) OnClose() {
	if a.gate.AgentChanRPC != nil {
		a.gate.AgentChanRPC.Go("CloseAgent", a)
	}
}

func (a *agent) RespMsg(ctx uint32, msg interface{}) {
	if a.gate.Processor != nil {
		data, err := a.gate.Processor.Marshal(ctx, msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *agent) SendMsg(msg interface{}) {
	if a.gate.Processor != nil {
		data, err := a.gate.Processor.Marshal(0, msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}
