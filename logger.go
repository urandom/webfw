package webfw

import (
	"io"
	"log"
)

// The logger interface provides some common methods for outputting messages.
// It may be used to exchange the default log.Logger error logger with another
// provider.
type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})

	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})

	Info(v ...interface{})
	Infof(format string, v ...interface{})
	Infoln(v ...interface{})

	Debug(v ...interface{})
	Debugf(format string, v ...interface{})
	Debugln(v ...interface{})
}

type StandardLogger struct {
	*log.Logger
}

func NewStandardLogger(out io.Writer, prefix string, flag int) StandardLogger {
	return StandardLogger{Logger: log.New(out, prefix, flag)}
}

func (st StandardLogger) Info(v ...interface{}) {
	st.Print(v...)
}

func (st StandardLogger) Infof(format string, v ...interface{}) {
	st.Printf(format, v...)
}

func (st StandardLogger) Infoln(v ...interface{}) {
	st.Println(v...)
}

func (st StandardLogger) Debug(v ...interface{}) {
	st.Print(v...)
}

func (st StandardLogger) Debugf(format string, v ...interface{}) {
	st.Printf(format, v...)
}

func (st StandardLogger) Debugln(v ...interface{}) {
	st.Println(v...)
}
