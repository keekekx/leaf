package util

import "fmt"

type ErrorInfo struct {
	Code interface{}
	Debug string
}

func (e *ErrorInfo) Error() string {
	return fmt.Sprintf("Code:%s,Debug:%s", e.Code, e.Debug)
}

func Assert(check bool, code interface{}, debug string) {
	if !check {
		panic(&ErrorInfo{
			Code : code,
			Debug: debug,
		})
	}
}
