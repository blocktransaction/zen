package log

type option struct {
	serivceName    string
	logFilePath    string
	logFileName    string
	logFileMaxSize int
	logFileMaxAge  int
}

type Option func(*option)

func WithSerivceName(serivceName string) Option {
	return func(o *option) {
		o.serivceName = serivceName
	}
}

func WithLogFilePath(logFilePath string) Option {
	return func(o *option) {
		o.logFilePath = logFilePath
	}
}

func WithLogFileName(logFileName string) Option {
	return func(o *option) {
		o.logFileName = logFileName
	}
}

func WithLogFileMaxSize(logFileMaxSize int) Option {
	return func(o *option) {
		o.logFileMaxSize = logFileMaxSize
	}
}

func WithLogLogFileMaxAge(logFileMaxAge int) Option {
	return func(o *option) {
		o.logFileMaxAge = logFileMaxAge
	}
}
