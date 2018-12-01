package kitc

import (
	"fmt"
)

type KitcError struct {
	head string
	text string
}

func Err(format string, v ...interface{}) *KitcError {
	return &KitcError{
		head: "kitc:",
		text: fmt.Sprintf(format, v...),
	}
}

func (e *KitcError) Error() string {
	return e.head + " " + e.text
}
