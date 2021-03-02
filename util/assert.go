package util

import "fmt"

type ErrorInfo struct {
	Code interface{}
	Debug string
	Kick bool
}

func (e *ErrorInfo) Error() string {
	return fmt.Sprintf("Code:%s,Debug:%s", e.Code, e.Debug)
}

func NewError(code interface{}, debug string) *ErrorInfo {
	return &ErrorInfo{
		Code : code,
		Debug: debug,
	}
}

func Assert(check bool, code interface{}, debug string) {
	if !check {
		panic(&ErrorInfo{
			Code : code,
			Debug: debug,
		})
	}
}

func AssertKick(check bool, code interface{}, debug string) {
	if !check {
		panic(&ErrorInfo{
			Code : code,
			Debug: debug,
			Kick: true,
		})
	}
}