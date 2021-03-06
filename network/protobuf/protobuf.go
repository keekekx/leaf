package protobuf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/keekekx/leaf/chanrpc"
	"github.com/keekekx/leaf/log"
	"reflect"
)

// -------------------------
// | id | protobuf message |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[uint32]*MsgInfo
	msgID        map[reflect.Type]uint32
}

type MsgInfo struct {
	msgType       reflect.Type
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
}

type MsgHandler func([]interface{}) (interface{}, error)

type MsgRaw struct {
	msgID      uint32
	msgRawData []byte
}

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgInfo = make(map[uint32]*MsgInfo)
	p.msgID = make(map[reflect.Type]uint32)
	return p
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg proto.Message, nid uint32) {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("protobuf message pointer required")
	}
	if _, ok := p.msgID[msgType]; ok {
		log.Fatal("message %s is already registered", msgType)
	}

	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[nid] = i
	p.msgID[msgType] = nid
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg proto.Message, msgRouter *chanrpc.Server) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		log.Fatal("message %s not registered", msgType)
	}

	p.msgInfo[id].msgRouter = msgRouter
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(msg proto.Message, msgHandler MsgHandler) {
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		log.Fatal("message %s not registered", msgType)
	}

	p.msgInfo[id].msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRawHandler(id uint32, msgRawHandler MsgHandler) {
	if _, ok := p.msgInfo[id]; !ok {
		log.Fatal("message id %v not registered", id)
	}

	p.msgInfo[id].msgRawHandler = msgRawHandler
}

// goroutine safe
func (p *Processor) Route(msg interface{}, userData interface{}) (interface{}, error) {
	// raw
	if msgRaw, ok := msg.(MsgRaw); ok {
		if _, ok := p.msgInfo[msgRaw.msgID]; !ok {
			return nil, fmt.Errorf("message id %v not registered", msgRaw.msgID)
		}

		i := p.msgInfo[msgRaw.msgID]
		if i.msgRawHandler != nil {
			return i.msgRawHandler([]interface{}{msgRaw.msgID, msgRaw.msgRawData, userData})
		}
		return nil, nil
	}

	// protobuf
	msgType := reflect.TypeOf(msg)
	id, ok := p.msgID[msgType]
	if !ok {
		return nil, fmt.Errorf("message %s not registered", msgType)
	}
	i := p.msgInfo[id]
	if i.msgHandler != nil {
		return i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		return i.msgRouter.Dispatch(msgType, msg, userData)
	}
	return nil, nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte) (uint32, interface{}, error) {
	if len(data) < 8 {
		return 0, nil, errors.New("protobuf data too short")
	}

	var ctx uint32
	if p.littleEndian {
		ctx = binary.LittleEndian.Uint32(data)
	} else {
		ctx = binary.BigEndian.Uint32(data)
	}

	// id
	var id uint32
	if p.littleEndian {
		id = binary.LittleEndian.Uint32(data[4:])
	} else {
		id = binary.BigEndian.Uint32(data[4:])
	}
	if _, ok := p.msgInfo[id]; !ok {
		return ctx, nil, fmt.Errorf("message id %v not registered", id)
	}

	// msg
	i := p.msgInfo[id]
	if i.msgRawHandler != nil {
		return ctx, MsgRaw{id, data[8:]}, nil
	} else {
		msg := reflect.New(i.msgType.Elem()).Interface()
		return ctx, msg, proto.UnmarshalMerge(data[8:], msg.(proto.Message))
	}
}

// goroutine safe
func (p *Processor) Marshal(_ctx uint32, msg interface{}) ([][]byte, error) {
	msgType := reflect.TypeOf(msg)

	// id
	_id, ok := p.msgID[msgType]
	if !ok {
		err := fmt.Errorf("message %s not registered", msgType)
		return nil, err
	}

	ctx := make([]byte, 4)
	if p.littleEndian {
		binary.LittleEndian.PutUint32(ctx, _ctx)
	} else {
		binary.BigEndian.PutUint32(ctx, _ctx)
	}

	id := make([]byte, 4)
	if p.littleEndian {
		binary.LittleEndian.PutUint32(id, _id)
	} else {
		binary.BigEndian.PutUint32(id, _id)
	}

	// data
	data, err := proto.Marshal(msg.(proto.Message))
	return [][]byte{ctx, id, data}, err
}

// goroutine safe
func (p *Processor) Range(f func(id uint32, t reflect.Type)) {
	for id, i := range p.msgInfo {
		f(id, i.msgType)
	}
}
