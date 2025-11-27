package constant

const (
	Prod = "prod"
	Test = "test"
	Env  = "env"

	Basicdata     = "basicdata"
	Authorization = "authorization"
	XSource       = "x-source" //来源
	XSourceValue  = ""
	Language      = "language" //语言
	UserId        = "userid"
	TraceId       = "traceID"
)

type ctxKey string

const (
	CtxKeyUserId ctxKey = "userId"
	CtxKeyEnv    ctxKey = "env"
	CtxKeyLang   ctxKey = "lang"
	CtxKeyTrace  ctxKey = "traceID"
)
