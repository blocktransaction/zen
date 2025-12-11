package logx

import (
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	instance *zap.Logger
	once     sync.Once
)

// ErrorWrapper 用于日志包装
type ErrorWrapper struct {
	Err      error
	Code     int
	Stack    string
	Metadata map[string]interface{}
}

//  构建 ErrorWrapper
func NewErrorWrapper(err error, code int) *ErrorWrapper {
	return &ErrorWrapper{
		Err:   err,
		Code:  code,
		Stack: string(debug.Stack()),
	}
}

//  初始化 logger
func Init(service, logPath string, isProd bool) *zap.Logger {
	once.Do(func() {
		instance = newLogger(service, logPath, isProd)
	})
	return instance
}

func Logger() *zap.Logger {
	return instance
}

func newLogger(service, logPath string, isProd bool) *zap.Logger {
	if service == "" {
		service = "service"
	}
	if logPath == "" {
		logPath = "."
	}

	// 按日期目录分割
	dateDir := time.Now().Format("2006-01-02")
	dir := path.Join(logPath, service, dateDir)
	_ = os.MkdirAll(dir, 0755)

	fileWriter := &lumberjack.Logger{
		Filename:  path.Join(dir, "app.log"),
		MaxSize:   200, // MB
		MaxAge:    30,  // 天
		Compress:  true,
		LocalTime: true,
	}

	writer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(fileWriter), zapcore.AddSync(os.Stdout))
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
	}
	var encoder zapcore.Encoder
	atomicLevel := zap.NewAtomicLevel()
	if isProd {
		atomicLevel.SetLevel(zap.InfoLevel)
		encoder = zapcore.NewJSONEncoder(encCfg)
	} else {
		atomicLevel.SetLevel(zap.DebugLevel)
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	core := zapcore.NewCore(encoder, writer, atomicLevel)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
