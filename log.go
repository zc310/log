package log

import (
	"fmt"

	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type InfoLogger interface {
	// Info logs a non-error message.  This is behaviorally akin to fmt.Print.
	Print(a ...interface{})

	// Infof logs a formatted non-error message.
	Printf(format string, a ...interface{})
}

// Logger represents the ability to log messages, both errors and not.
type Logger interface {
	// All Loggers implement InfoLogger.  Calling InfoLogger methods directly on
	// a Logger value is equivalent to calling them on a V(0) InfoLogger.  For
	// example, logger.Info() produces the same result as logger.V(0).Info.
	InfoLogger

	// Error logs a error message.  This is behaviorally akin to fmt.Print.
	Error(args ...interface{})

	// Errorf logs a formatted error message.
	Errorf(format string, args ...interface{})

	// V returns an InfoLogger value for a specific verbosity level.  A higher
	// verbosity level means a log message is less important.
	V(level int) InfoLogger

	// NewWithPrefix returns a Logger which prefixes all messages.
	NewWithPrefix(prefix string) Logger
	WithFormatter(v Formatter) Logger
}

var (
	logLevel   int
	Default    Logger
	logOut     = make(map[string]*Write)
	lock       sync.Mutex
	logPath    string = filepath.Join(os.TempDir(), "logs")
	maxSize    int    = 30
	maxBackups int    = 0
	maxAge     int    = 7
	compress   bool   = true
)

type Write struct {
	w io.Writer
}

type Log struct {
	info      io.Writer
	err       io.Writer
	pool      *sync.Pool
	Prefix    string
	Level     int
	Formatter Formatter
}
type Entry struct {
	// Time at which the log entry was created
	Time string `json:"time"`

	// Message passed to  Info,  Error
	Message interface{} `json:"msg"`
}
type Formatter interface {
	Format(*Entry, io.Writer) error
}

func (p Write) Write(b []byte) (n int, err error) {
	return p.w.Write(b)
}
func SetDefault(v Logger) {
	Default = v
}
func SetLevel(v int) {
	logLevel = v
}
func SetPath(s string) {
	logPath = s
}
func (p Log) V(level int) InfoLogger {
	return V(level)
}

func (p Log) NewWithPrefix(prefix string) Logger {
	return NewWithPrefix(prefix)
}

func init() {
	Default = New(0, "")
}

func New(level int, prefix string) *Log {
	p := &Log{Prefix: prefix, Level: level}
	p.pool = &sync.Pool{
		New: func() interface{} {
			return new(Entry)
		},
	}
	p.Formatter = &TextFormatter{}

	p.info = getOut(level, "info", prefix)
	p.err = getOut(0, "error", prefix)
	return p
}
func (p *Log) WithFormatter(v Formatter) Logger {
	p.Formatter = v
	return p
}
func (p Log) Outputf(w io.Writer, format string, a ...interface{}) {
	if logLevel >= p.Level {
		entry := p.pool.Get().(*Entry)
		entry.Time = time.Now().Format(time.RFC3339)
		entry.Message = fmt.Sprintf(format, a...)

		err := p.Formatter.Format(entry, w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
		}

		p.pool.Put(entry)
	}
}
func (p Log) Output(w io.Writer, a ...interface{}) {
	if logLevel >= p.Level {
		entry := p.pool.Get().(*Entry)
		entry.Time = time.Now().Format(time.RFC3339)

		b := make([]interface{}, len(a))
		for i, arg := range a {
			switch v := arg.(type) {
			case error:
				b[i] = v.Error()
			case fmt.Stringer:
				b[i] = v.String()
			case string:
				b[i] = v
			default:
				b[i] = arg
			}
		}
		if len(b) == 1 {
			entry.Message = b[0]
		} else {
			entry.Message = b
		}

		err := p.Formatter.Format(entry, w)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
		}
		p.pool.Put(entry)
	}
}
func (p Log) Printf(format string, a ...interface{}) {
	p.Outputf(p.info, format, a...)
}
func (p Log) Print(a ...interface{}) {
	p.Output(p.info, a...)
}
func (p Log) Errorf(format string, a ...interface{}) {
	p.Outputf(p.err, format, a...)
}

func (p Log) Error(args ...interface{}) {
	p.Output(p.err, args...)
}

func Close() {
	for _, w := range logOut {
		if b, ok := w.w.(*lumberjack.Logger); ok {
			b.Close()
		}
	}
}

func Rotate() {
	for _, w := range logOut {
		if b, ok := w.w.(*lumberjack.Logger); ok {
			b.Rotate()
		}
	}
}

func newOut(level int, name, prefix string) *Write {
	var filename []string
	filename = append(filename, logPath)
	filename = append(filename, strings.Split(prefix, ".")...)
	if level == 0 {
		filename = append(filename, fmt.Sprintf("%s.log", name))
	} else {
		filename = append(filename, fmt.Sprintf("%s_%.2d.log", name, level))
	}
	return &Write{&lumberjack.Logger{
		Filename:   filepath.Join(filename...),
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}}
}
func SetWriter(level int, prefix string, info, error io.Writer) {
	lock.Lock()
	defer lock.Unlock()
	var (
		key string
		out *Write
	)

	key = fmt.Sprintf("%d_info_%s", level, prefix)
	if out = logOut[key]; out != nil {
		out.w = info
	} else {
		logOut[key] = &Write{info}
	}
	key = fmt.Sprintf("%d_error_%s", level, prefix)
	if out = logOut[key]; out != nil {
		out.w = error
	} else {
		logOut[key] = &Write{error}
	}
}
func getOut(level int, name, prefix string) *Write {
	key := fmt.Sprintf("%d_%s_%s", level, name, prefix)
	if out, ok := logOut[key]; ok {
		return out
	}
	lock.Lock()
	defer lock.Unlock()
	logOut[key] = newOut(level, name, prefix)
	return logOut[key]

}
func Infof(format string, a ...interface{}) {
	Default.Printf(format, a...)
}
func Info(a ...interface{}) {
	Default.Print(a...)
}
func Errorf(format string, a ...interface{}) {
	Default.Errorf(format, a...)
}
func Error(a ...interface{}) {
	Default.Error(a...)
}
func Printf(format string, a ...interface{}) {
	Default.Printf(format, a...)
}
func Print(a ...interface{}) {
	Default.Print(a...)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func Fatal(a ...interface{}) {
	Default.Error(a...)
	panic(fmt.Sprint(a...))
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func Fatalf(format string, a ...interface{}) {
	Default.Errorf(format, a...)
	panic(fmt.Sprintf(format, a...))
}
func V(level int) InfoLogger {
	return New(level, "")
}

func NewWithPrefix(prefix string) Logger {
	return New(0, prefix)
}
