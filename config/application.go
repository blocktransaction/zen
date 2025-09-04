package config

type Application struct {
	Env                 string
	LogName             string
	LogFilePath         string
	LogFileName         string
	LogFileMaxSize      int
	LogFileMaxAge       int
	I18nFilePath        string
	I18nSupportLanguage []string
	DefaultLang         string
	TemplateFile        string
	JwtExpiresAt        int64
	UserExpiresAt       int64
	MaxUploadImageNum   int
}

var ApplicationConfig = new(Application)
