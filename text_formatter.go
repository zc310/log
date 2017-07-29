package log

import (
	"encoding/json"
	"github.com/valyala/bytebufferpool"
	"io"
)

type TextFormatter struct{}

func (f *TextFormatter) Format(entry *Entry, w io.Writer) error {
	b, err := json.Marshal(entry.Message)
	if err != nil {
		return err
	}
	buf := bytebufferpool.Get()
	buf.Write([]byte(entry.Time))
	buf.Write([]byte("\t"))
	buf.Write(b)
	buf.Write([]byte("\n"))
	_, err = w.Write(buf.B)
	bytebufferpool.Put(buf)
	return err
}
