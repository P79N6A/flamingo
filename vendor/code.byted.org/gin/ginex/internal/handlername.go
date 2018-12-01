package internal

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

var (
	handlerNames = make(map[uintptr]string)
)

func SetHandlerName(handler gin.HandlerFunc, name string) {
	handlerNames[reflect.ValueOf(handler).Pointer()] = name
}

func GetHandlerName(handler gin.HandlerFunc) string {
	return handlerNames[reflect.ValueOf(handler).Pointer()]
}
