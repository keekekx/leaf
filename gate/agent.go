package gate

import (
	"net"
)

type Agent interface {
	RespMsg(ctx uint32, msg interface{})
	SendMsg(msg interface{})
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()
	UserData() interface{}
	SetUserData(data interface{})
}
