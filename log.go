package zlog

import (
	"fmt"
	"log"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	LEVEL_FLAGS = [...]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL", "PUBLIC"}
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARNING
	ERROR
	FATAL
	PUBLIC
)

const tunnel_size_default = 1024
const buffer_bytes_cnt = 1024

type Record struct {
	time   time.Time
	code   string
	line   int
	info   string
	level  int
	args   []interface{}
	hight  bool
	fields []Field
}

type textEncoder struct {
	bytes []byte
	level int
}

func (enc *textEncoder) truncate() {
	enc.bytes = enc.bytes[:0]
}

func (enc *textEncoder) Write(p []byte) (int, error) {
	enc.bytes = append(enc.bytes, p...)
	return len(p), nil
}

var textPool = sync.Pool{New: func() interface{} {
	return &textEncoder{
		bytes: make([]byte, 0, buffer_bytes_cnt),
	}
}}

func (r *Record) Bytes(enc *textEncoder) {
	enc.bytes = append(enc.bytes, '[')
	enc.bytes = r.time.AppendFormat(enc.bytes, logger_default.layout)
	enc.bytes = append(enc.bytes, "] ["...)
	enc.bytes = append(enc.bytes, LEVEL_FLAGS[r.level]...)
	enc.bytes = append(enc.bytes, "] ["...)
	enc.bytes = append(enc.bytes, r.code...)
	enc.bytes = append(enc.bytes, ':')
	enc.bytes = strconv.AppendInt(enc.bytes, int64(r.line), 10)
	enc.bytes = append(enc.bytes, "] "...)
	meta := " "
	if r.level == PUBLIC {
		meta = "||"
	}
	if r.hight {
		enc.bytes = append(enc.bytes, r.info...)
		for _, field := range r.fields {
			enc.bytes = append(enc.bytes, meta...)
			enc.bytes = append(enc.bytes, field.key...)
			enc.bytes = append(enc.bytes, '=')
			enc.bytes = field.WriteValue(enc.bytes)
		}
	} else {
		if len(r.args) == 0 {
			enc.bytes = append(enc.bytes, r.info...)
		} else {
			fmt.Fprintf(enc, r.info, r.args...)
		}
	}
	enc.bytes = append(enc.bytes, '\n')
}

type Writer interface {
	Init() error
	Write(*textEncoder) error
}

type Rotater interface {
	Rotate() error
	SetPathPattern(string) error
}

type Flusher interface {
	Flush() error
}

type Logger struct {
	writers []Writer
	tunnel  chan *textEncoder
	level   int
	c       chan bool
	layout  string
}

func NewLogger() *Logger {
	l := new(Logger)
	l.writers = make([]Writer, 0, 2)
	l.tunnel = make(chan *textEncoder, tunnel_size_default)
	l.c = make(chan bool, 1)
	l.level = DEBUG
	l.layout = "2006-01-02T15:04:05.000+0800"

	go boostrapLogWriter(l)

	return l
}

func (l *Logger) Register(w Writer) {
	if err := w.Init(); err != nil {
		panic(err)
	}
	l.writers = append(l.writers, w)
}

func (l *Logger) SetLevel(lvl int) {
	l.level = lvl
}

func (l *Logger) SetLayout(layout string) {
	l.layout = layout
}

func (l *Logger) Public(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(PUBLIC, fmt, args...)
}

func (l *Logger) Trace(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(TRACE, fmt, args...)
}

func (l *Logger) Debug(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(DEBUG, fmt, args...)
}

func (l *Logger) Warn(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(WARNING, fmt, args...)
}

func (l *Logger) Info(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(INFO, fmt, args...)
}

func (l *Logger) Error(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(ERROR, fmt, args...)
}

func (l *Logger) Fatal(fmt string, args ...interface{}) {
	l.deliverRecordToWriter(FATAL, fmt, args...)
}

func (l *Logger) Close() {
	close(l.tunnel)
	<-l.c

	for _, w := range l.writers {
		if f, ok := w.(Flusher); ok {
			if err := f.Flush(); err != nil {
				log.Println(err)
			}
		}
	}
}

func (l *Logger) deliverRecordToWriter(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	// source code, file and line num
	_, file, line, ok := runtime.Caller(2)
	r := &Record{}
	r.hight = false
	r.info = format
	if ok {
		r.code = path.Base(file)
		r.line = line
	}
	r.time = time.Now()
	r.level = level
	r.args = args

	enc := textPool.Get().(*textEncoder)
	enc.truncate()
	enc.level = r.level
	r.Bytes(enc)

	l.tunnel <- enc
}

func (l *Logger) deliverRecordToWriterHight(level int, with string, fields ...Field) {
	if level < l.level {
		return
	}
	// source code, file and line num
	_, file, line, ok := runtime.Caller(2)
	r := &Record{}
	r.hight = true
	r.info = with
	if ok {
		r.code = path.Base(file)
		r.line = line
	}
	r.time = time.Now()
	r.level = level
	r.fields = fields

	enc := textPool.Get().(*textEncoder)
	enc.truncate()
	enc.level = r.level
	r.Bytes(enc)

	l.tunnel <- enc
}

func boostrapLogWriter(logger *Logger) {
	if logger == nil {
		panic("logger is nil")
	}

	var (
		enc *textEncoder
		ok  bool
	)

	if enc, ok = <-logger.tunnel; !ok {
		logger.c <- true
		return
	}

	for _, w := range logger.writers {
		if err := w.Write(enc); err != nil {
			log.Println(err)
		}
	}

	flushTimer := time.NewTimer(time.Millisecond * 500)
	rotateTimer := time.NewTimer(time.Second * 10)

	for {
		select {
		case enc, ok = <-logger.tunnel:
			if !ok {
				logger.c <- true
				return
			}

			for _, w := range logger.writers {
				if err := w.Write(enc); err != nil {
					log.Println(err)
				}
			}

		case <-flushTimer.C:
			for _, w := range logger.writers {
				if f, ok := w.(Flusher); ok {
					if err := f.Flush(); err != nil {
						log.Println(err)
					}
				}
			}
			flushTimer.Reset(time.Millisecond * 1000)

		case <-rotateTimer.C:
			for _, w := range logger.writers {
				if r, ok := w.(Rotater); ok {
					if err := r.Rotate(); err != nil {
						log.Println(err)
					}
				}
			}
			rotateTimer.Reset(time.Second * 10)
		}
	}
}

// default
var (
	logger_default *Logger
)

func SetLevel(lvl int) {
	logger_default.level = lvl
}

func SetLayout(layout string) {
	logger_default.layout = layout
}

func Public(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(PUBLIC, fmt, args...)
}

func Trace(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(TRACE, fmt, args...)
}

func Debug(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(DEBUG, fmt, args...)
}

func Warn(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(WARNING, fmt, args...)
}

func Info(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(INFO, fmt, args...)
}

func Error(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(ERROR, fmt, args...)
}

func Fatal(fmt string, args ...interface{}) {
	logger_default.deliverRecordToWriter(FATAL, fmt, args...)
}
func HighInfo(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(INFO, with, fields...)
}
func HighWarn(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(WARNING, with, fields...)
}
func HighError(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(ERROR, with, fields...)
}
func HighPublic(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(PUBLIC, with, fields...)
}
func HighDebug(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(DEBUG, with, fields...)
}
func HighTrace(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(TRACE, with, fields...)
}
func HighFatal(with string, fields ...Field) {
	logger_default.deliverRecordToWriterHight(FATAL, with, fields...)
}

func Register(w Writer) {
	logger_default.Register(w)
}

func (l *Logger) RegisterWithFile(filepath, rotatelogpath string, level int) {
	w := NewFileWriter()
	w.SetFileName(filepath)
	w.SetPathPattern(rotatelogpath)
	w.SetLogLevelFloor(level)
	l.Register(w)
}

func NewLoggerWithFile(filepath, rotatelogpath string, level int) *Logger {
	l := NewLogger()
	l.RegisterWithFile(filepath, rotatelogpath, level)
	return l
}

func Close() {
	logger_default.Close()
}

func init() {
	logger_default = NewLogger()
}
