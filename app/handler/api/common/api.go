package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/internal/i18n"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// 统一响应结构
type Response struct {
	Code interface{} `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 分页响应结构
type PaginationResponse struct {
	Count     interface{} `json:"count"`
	List      interface{} `json:"list"`
	PageSize  int         `json:"pageSize"`
	PageIndex int         `json:"pageIndex"`
}

type EmptyStruct struct{}

// api
type Api struct {
	ginContext    *gin.Context    //gin上下文
	commonContext context.Context //公共上下文
	Errors        error           //错误信息

	//api header info
	language string //语言
	env      string
	userId   int64
	validate *validator.Validate
}

// Bind 参数校验
func (a *Api) Bind(d interface{}, bindings ...binding.Binding) *Api {
	// 如果没有指定绑定方式，则根据结构体标签自动推断
	if len(bindings) == 0 {
		bindings = constructor.GetBindingForGin(d)
	}

	// 执行绑定操作
	for _, b := range bindings {
		var err error
		if b == nil {
			// 当绑定为nil时，尝试绑定URI参数
			err = a.ginContext.ShouldBindUri(d)
		} else {
			// 使用指定的绑定方式
			err = a.ginContext.ShouldBindWith(d, b)
		}

		if err != nil {
			// 处理特殊错误情况
			if errors.Is(err, io.EOF) {
				a.AddError(errors.New("input null"))
			} else {
				a.AddError(err)
			}
			// 绑定失败时直接返回，不继续执行验证
			return a
		}
	}

	// 只有在绑定成功后才进行结构体验证
	if a.validate != nil {
		if err := a.validate.Struct(d); err != nil {
			a.AddError(err)
		}
	}

	return a
}

// 添加error
func (a *Api) AddError(err error) {
	if err == nil {
		return
	}
	a.Errors = errors.Join(a.Errors, err)
}

// 上下文 处理对应公共头参数
func (a *Api) WithContext(c *gin.Context) *Api {
	a.ginContext = c

	a.commonContext = context.Background()
	a.commonContext = context.WithValue(a.commonContext, constant.UserIdKey, a.defaultUserId())
	a.commonContext = context.WithValue(a.commonContext, constant.EnvKey, a.defaultEnv())
	a.commonContext = context.WithValue(a.commonContext, constant.LangKey, a.defaultLanguage())
	a.commonContext = context.WithValue(a.commonContext, constant.TraceIdKey, c.GetString(constant.TraceId))

	a.validate = validator.New(validator.WithRequiredStructEnabled())

	return a
}

func (a *Api) defaultUserId() int64 {
	userId := a.ginContext.GetInt64(constant.UserId)
	a.userId = userId
	return userId
}

func (a *Api) defaultLanguage() string {
	lang := a.ginContext.GetHeader(constant.Language)
	if lang == "" {
		lang = i18n.En
	}
	a.language = lang
	return lang
}

func (a *Api) defaultEnv() string {
	env := a.ginContext.GetHeader(constant.Env)
	if env == "" {
		env = constant.Test
	}
	a.env = env
	return env
}

// 获取上下文
func (a *Api) GetContext() context.Context {
	return a.commonContext
}

// 获取当前语言
func (a *Api) GetLang() string {
	return a.language
}

// 获取当前环境
func (a *Api) GetEnv() string {
	return a.env
}

// 获取用户id
func (a *Api) GetUserId() int64 {
	return a.userId
}

// 统一的响应发送方法
func (a *Api) sendResponse(code interface{}, msg string, data interface{}) {
	// 处理nil数据
	if data == nil {
		data = EmptyStruct{}
	}

	// 处理数组、指针、map、接口等nil数据
	if v := reflect.ValueOf(data); v.IsValid() {
		switch v.Kind() {
		case reflect.Slice:
			if v.IsNil() {
				data = make([]EmptyStruct, 0)
			}
		case reflect.Ptr, reflect.Map, reflect.Interface:
			if v.IsNil() {
				data = EmptyStruct{}
			}
		}
	}

	response := Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}

	a.ginContext.JSON(http.StatusOK, response)
	a.ginContext.Abort()
}

// 成功响应
func (a *Api) Success(msg string, data interface{}) {
	a.sendResponse(0, msg, data)
}

// 带语言代码的成功响应
func (a *Api) SuccessWithCode(code string, data interface{}) {
	msg := i18n.GetManager().GetMessage(code)
	a.sendResponse(0, msg, data)
}

// 分页成功响应
func (a *Api) SuccessWithPagination(msg string, count, data interface{}, pageSize, pageIndex int) {
	paginationData := PaginationResponse{
		Count:     count,
		List:      data,
		PageSize:  pageSize,
		PageIndex: pageIndex,
	}
	a.sendResponse(0, msg, paginationData)
}

// 错误响应
func (a *Api) Error(code string) {
	msg := i18n.GetManager().GetMessage(code)
	a.sendResponse(parseErrorCodeFlexible(code), msg, EmptyStruct{})
}

// 带自定义消息的错误响应
func (a *Api) ErrorWithMsg(code, msg string) {
	a.sendResponse(parseErrorCodeFlexible(code), msg, EmptyStruct{})
}

// 带语言代码和自定义消息的错误响应
func (a *Api) ErrorWithCodeAndMsg(code, customMsg string) {
	baseMsg := i18n.GetManager().GetMessage(code)
	msg := fmt.Sprintf("%s\nerror: %s", baseMsg, customMsg)
	a.sendResponse(parseErrorCodeFlexible(code), msg, EmptyStruct{})
}

// 带参数的错误响应
func (a *Api) ErrorWithParams(code string, params ...interface{}) {
	baseMsg := i18n.GetManager().GetMessage(code)
	msg := fmt.Sprintf(baseMsg, params...)
	a.sendResponse(parseErrorCodeFlexible(code), msg, EmptyStruct{})
}

// 解析错误代码；数字则返回数值，否则返回原字符串
func parseErrorCodeFlexible(code string) interface{} {
	if parsed, err := strconv.Atoi(code); err == nil {
		return parsed
	}
	return code
}
