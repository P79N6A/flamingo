package buffer_pool

import (
	"bytes"
	"sync"
)

var bp sync.Pool

func init() {
	bp.New = func() interface{} {
		return &bytes.Buffer{}
	}
}

func Get() *bytes.Buffer {
	return bp.Get().(*bytes.Buffer)
}

func Put(b *bytes.Buffer) {
	bp.Put(b)
}
