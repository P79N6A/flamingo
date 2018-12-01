package logs

import (
	"bytes"

	"code.byted.org/gopkg/logfmt"
)

type LogEncoder struct {
	*logfmt.Encoder
	buf bytes.Buffer
}

func (le *LogEncoder) Reset() {
	le.Encoder.Reset()
	le.buf.Reset()
}
