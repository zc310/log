package log

import (
	"encoding/json"
	"io"
)

type JSONFormatter struct{}

func (p *JSONFormatter) Format(entry *Entry, w io.Writer) error {
	return json.NewEncoder(w).Encode(entry)
}
