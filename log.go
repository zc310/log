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
	pid        = os.Getpid()
	program    = filepath.Base(os.Args[0])
	logLevel   int
	log        Logger
	logOut     = make(map[string]*Write)
	lock       sync.Mutex
	logPath    string = filepath.Join(os.TempDir(), "logs")
	maxSize    int    = 30
	maxBackups int    = 0
	maxAge     int    = 7
)

type Write struct {
	out io.Writer
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
	return p.out.Write(b)
}
func (p Write) Rotate() {
	if w, ok := p.out.(*lumberjack.Logger); ok {
		w.Rotate()
	}
}
func (p Write) Close() {
	if w, ok := p.out.(*lumberjack.Logger); ok {
		w.Close()
	}
}
func SetDefault(v Logger) {
	log = v
}
func SetLevel(v int) {
	logLevel = v
}

// SetPath Set the log save path
func SetPath(s string) {
	for _, w := range logOut {
		if lj, ok := w.out.(*lumberjack.Logger); ok {
			w.out = &lumberjack.Logger{
				Filename:   filepath.Join(s, strings.TrimPrefix(lj.Filename, logPath)),
				MaxSize:    lj.MaxSize,
				MaxBackups: lj.MaxBackups,
				MaxAge:     lj.MaxAge,
				Compress:   lj.Compress,
			}
			lj.Close()
		}
	}
	logPath = s
}
func (p Log) V(level int) InfoLogger {
	return V(level)
}

func (p Log) NewWithPrefix(prefix string) Logger {
	return NewWithPrefix(prefix)
}

func init() {
	SetDefault(New(0, ""))
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

// Printf calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Printf.
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

// Close closes the all logfile.
func Close() {
	for _, w := range logOut {
		w.Close()
	}
}

// Rotate closes All files, moves it aside with a timestamp in the name,
func Rotate() {
	for _, w := range logOut {
		w.Rotate()
	}
}

func newOut(level int, name, prefix string) *Write {
	var filename []string
	filename = append(filename, logPath)
	filename = append(filename, strings.Split(prefix, ".")...)
	if level == 0 {
		filename = append(filename, fmt.Sprintf("%s_%s-%d.log", program, name, pid))
	} else {
		filename = append(filename, fmt.Sprintf("%s_%s_%.2d-%d.log", program, name, level, pid))
	}
	return &Write{&lumberjack.Logger{
		Filename:   filepath.Join(filename...),
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   true,
	}}
}

// SetOutput sets the output destination for the logger.
func SetOutput(level int, prefix string, info, error io.Writer) {
	lock.Lock()
	defer lock.Unlock()
	var (
		key string
		out *Write
	)

	key = fmt.Sprintf("%d_info_%s", level, prefix)
	if out = logOut[key]; out != nil {
		out.out = info
	} else {
		logOut[key] = &Write{info}
	}
	key = fmt.Sprintf("%d_error_%s", level, prefix)
	if out = logOut[key]; out != nil {
		out.out = error
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
	log.Printf(format, a...)
}
func Info(a ...interface{}) {
	log.Print(a...)
}
func Errorf(format string, a ...interface{}) {
	log.Errorf(format, a...)
}
func Error(a ...interface{}) {
	log.Error(a...)
}
func Printf(format string, a ...interface{}) {
	log.Printf(format, a...)
}
func Print(a ...interface{}) {
	log.Print(a...)
}

// Fatal is equivalent to l.Print() followed by a call to os.Exit(1).
func Fatal(a ...interface{}) {
	log.Error(a...)
	panic(fmt.Sprint(a...))
}

// Fatalf is equivalent to l.Printf() followed by a call to os.Exit(1).
func Fatalf(format string, a ...interface{}) {
	log.Errorf(format, a...)
	panic(fmt.Sprintf(format, a...))
}
func V(level int) InfoLogger {
	return New(level, "")
}

func NewWithPrefix(prefix string) Logger {
	return New(0, prefix)
}
