package util

import (
	"fmt"
)

type ErrorInfo struct {
	Code interface{}
	Debug string
	Kick bool
}

func (e *ErrorInfo) Error() string {
	return fmt.Sprintf("Code:%s,Debug:%s", e.Code, e.Debug)
}