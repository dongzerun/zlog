package zlog

import (
	"fmt"
	"os"
)

type ConsoleWriter struct {
}

func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{}
}

func (w *ConsoleWriter) Write(enc *textEncoder) error {
	fmt.Fprint(os.Stdout, enc.bytes)
	textPool.Put(enc)

	return nil
}

func (w *ConsoleWriter) Init() error {
	return nil
}
