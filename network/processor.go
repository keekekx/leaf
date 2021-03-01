package network

type Processor interface {
	// must goroutine safe
	Route(msg interface{}, userData interface{}) (interface{}, error)
	// must goroutine safe
	Unmarshal(data []byte) (uint32, interface{}, error)
	// must goroutine safe
	Marshal(int32 uint32, msg interface{}) ([][]byte, error)
}
