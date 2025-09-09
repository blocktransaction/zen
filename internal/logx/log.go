package logx

import (
	"bytes"
	"log"
	"os"
	"path"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var instance *zap.Logger

type lumberjackWriteSyncer struct {
	*lumberjack.Logger
	buf       *bytes.Buffer
	logChan   chan []byte
	closeChan chan interface{}
	maxSize   int
}

// log初始化
func NewLogger(opts ...Option) *zap.Logger {
	options := &option{}
	for _, o := range opts {
		o(options)
	}

	instance = options.newLogger()
	return instance
}

func Logger() *zap.Logger {
	return instance
}

// new log
func (o *option) newLogger() *zap.Logger {
	directory := path.Join(o.logFilePath, o.serivceName)
	writers := []zapcore.WriteSyncer{o.newRollingFile(directory)}
	writers = append(writers, os.Stdout)
	logger, _ := newZapLogger(true, zapcore.NewMultiWriteSyncer(writers...))
	zap.RedirectStdLog(logger)

	return logger
}

// 日志分割
func (o *option) newRollingFile(directory string) zapcore.WriteSyncer {
	if err := os.MkdirAll(directory, 0755); err != nil {
		log.Println("failed create log directory:", directory, ":", err)
		return nil
	}

	return newLumberjackWriteSyncer(&lumberjack.Logger{
		Filename:  path.Join(directory, o.logFileName),
		MaxSize:   o.logFileMaxSize, //megabytes
		MaxAge:    o.logFileMaxAge,  //days
		LocalTime: true,
		Compress:  true, //是否压缩
	})
}

// new log
func newZapLogger(isProduction bool, output zapcore.WriteSyncer) (*zap.Logger, *zap.AtomicLevel) {
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.NanosDurationEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		},
	}

	var encoder zapcore.Encoder
	dyn := zap.NewAtomicLevel()
	if isProduction {
		dyn.SetLevel(zap.InfoLevel)
		encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encCfg)
	} else {
		dyn.SetLevel(zap.DebugLevel)
		encCfg.EncodeLevel = zapcore.LowercaseColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	return zap.New(zapcore.NewCore(encoder, output, dyn), zap.AddCaller()), &dyn
}

// 写同步
func newLumberjackWriteSyncer(l *lumberjack.Logger) *lumberjackWriteSyncer {
	ws := &lumberjackWriteSyncer{
		Logger:    l,
		buf:       bytes.NewBuffer([]byte{}),
		logChan:   make(chan []byte, 5000),
		closeChan: make(chan interface{}),
		maxSize:   1024,
	}
	go ws.run()
	return ws
}

// 运行
func (l *lumberjackWriteSyncer) run() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if l.buf.Len() > 0 {
				l.sync()
			}
		case bs := <-l.logChan:
			_, err := l.buf.Write(bs)
			if err != nil {
				continue
			}
			if l.buf.Len() > l.maxSize {
				l.sync()
			}
		case <-l.closeChan:
			l.sync()
			return
		}
	}
}

// 写入
func (l *lumberjackWriteSyncer) Write(bs []byte) (int, error) {
	b := make([]byte, len(bs))
	copy(b, bs)
	l.logChan <- b
	return 0, nil
}

// 同步
func (l *lumberjackWriteSyncer) Sync() error {
	return l.sync()
}

func (l *lumberjackWriteSyncer) sync() error {
	defer l.buf.Reset()
	_, err := l.Logger.Write(l.buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// 停止
func (l *lumberjackWriteSyncer) Stop() {
	close(l.closeChan)
}
