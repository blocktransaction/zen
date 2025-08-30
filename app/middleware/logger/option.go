package logger

type logConfig struct {
	enableReqBody  bool     //允许打印请求body
	enableRespBody bool     //允许打印响应body
	maxBodySize    int      //最大body大小
	sensitiveKeys  []string //敏感字段列表
	onlyJSONBody   bool     //只打印 application/json
	guessJSON      bool     //可选：在没有 Content-Type 时通过 body 猜测 JSON
}

type Option func(*logConfig)

func WithEnableReqBody(enableReqBody bool) Option {
	return func(o *logConfig) {
		o.enableReqBody = enableReqBody
	}
}

func WithEnableRespBody(enableRespBody bool) Option {
	return func(o *logConfig) {
		o.enableRespBody = enableRespBody
	}
}

func WithMaxBodySize(maxBodySize int) Option {
	return func(o *logConfig) {
		o.maxBodySize = maxBodySize
	}
}

func WithSensitiveKeys(sensitiveKeys []string) Option {
	return func(o *logConfig) {
		o.sensitiveKeys = sensitiveKeys
	}
}

func WithOnlyJSONBody(onlyJSONBody bool) Option {
	return func(o *logConfig) {
		o.onlyJSONBody = onlyJSONBody
	}
}

func WithGuessJSON(guessJSON bool) Option {
	return func(o *logConfig) {
		o.guessJSON = guessJSON
	}
}
