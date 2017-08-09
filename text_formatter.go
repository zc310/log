package log

import (
	"encoding/json"
	"github.com/valyala/bytebufferpool"
	"io"
)

type TextFormatter struct {
	pool bytebufferpool.Pool
}

func (p *TextFormatter) Format(entry *Entry, w io.Writer) error {
	b, err := json.Marshal(entry.Message)
	if err != nil {
		return err
	}
	buf := p.pool.Get()
	buf.Write([]byte(entry.Time))
	buf.Write([]byte("\t"))
	buf.Write(b)
	buf.Write([]byte("\n"))
	_, err = w.Write(buf.B)
	p.pool.Put(buf)
	return err
}
