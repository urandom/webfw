package webfw

// The logger interface provides some of the output methods as the standard
// log.Logger object. It may be used to exchange the default log.Logger error
// logger with another provider.
type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Fatalln(v ...interface{})

	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}
