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

type CtxKey string

const (
	UserIdKey  CtxKey = "userId"
	EnvKey     CtxKey = "env"
	LangKey    CtxKey = "language"
	TraceIdKey CtxKey = "traceID"
)
