package common

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
)

const (
	_ uint8 = iota
	json
	xml
	yaml
	form
	query
)

var constructor = &bindConstructor{}

type bindConstructor struct {
	cache map[string][]uint8
	mux   sync.Mutex
}

func (e *bindConstructor) GetBindingForGin(d interface{}) []binding.Binding {
	// 获取类型名称用于缓存
	typeName := reflect.TypeOf(d).String()

	// 尝试从缓存获取绑定信息
	bs := e.getBinding(typeName)
	if bs == nil {
		// 缓存未命中，重新构建绑定信息
		bs = e.resolve(d)
		// 将结果存入缓存
		e.setBinding(typeName, bs)
	}

	// 将内部绑定类型转换为Gin绑定类型
	gbs := make([]binding.Binding, 0, len(bs))
	bindingMap := make(map[uint8]binding.Binding)

	for _, b := range bs {
		switch b {
		case json:
			bindingMap[json] = binding.JSON
		case xml:
			bindingMap[xml] = binding.XML
		case yaml:
			bindingMap[yaml] = binding.YAML
		case form:
			bindingMap[form] = binding.Form
		case query:
			bindingMap[query] = binding.Query
		default:
			bindingMap[0] = nil
		}
	}

	// 将map转换为slice，避免重复
	for bindingType := range bindingMap {
		gbs = append(gbs, bindingMap[bindingType])
	}

	return gbs
}

// resolve 解析结构体标签，确定需要的绑定类型
func (e *bindConstructor) resolve(d interface{}) []uint8 {
	bs := make([]uint8, 0, 8) // 预分配容量，避免频繁扩容
	qType := reflect.TypeOf(d).Elem()

	// 定义标签到绑定类型的映射
	tagToBinding := map[string]uint8{
		"json":  json,
		"xml":   xml,
		"yaml":  yaml,
		"form":  form,
		"query": query,
		"uri":   0, // 特殊处理
	}

	// 用于去重的map
	seenBindings := make(map[uint8]bool)

	for i := 0; i < qType.NumField(); i++ {
		field := qType.Field(i)
		tag := field.Tag

		// 检查所有可能的标签
		for tagName, bindingType := range tagToBinding {
			if _, ok := tag.Lookup(tagName); ok {
				if !seenBindings[bindingType] {
					bs = append(bs, bindingType)
					seenBindings[bindingType] = true
				}
			}
		}

		// 处理嵌套结构体（dive标签）
		if t, ok := tag.Lookup("binding"); ok && strings.Contains(t, "dive") {
			qValue := reflect.ValueOf(d)
			if fieldValue := qValue.Field(i); fieldValue.IsValid() {
				nestedBindings := e.resolve(fieldValue.Interface())
				for _, nestedBinding := range nestedBindings {
					if !seenBindings[nestedBinding] {
						bs = append(bs, nestedBinding)
						seenBindings[nestedBinding] = true
					}
				}
			}
		}

		if t, ok := tag.Lookup("validate"); ok && strings.Contains(t, "dive") {
			qValue := reflect.ValueOf(d)
			if fieldValue := qValue.Field(i); fieldValue.IsValid() {
				nestedBindings := e.resolve(fieldValue.Interface())
				for _, nestedBinding := range nestedBindings {
					if !seenBindings[nestedBinding] {
						bs = append(bs, nestedBinding)
						seenBindings[nestedBinding] = true
					}
				}
			}
		}
	}

	return bs
}

func (e *bindConstructor) getBinding(name string) []uint8 {
	e.mux.Lock()
	defer e.mux.Unlock()
	return e.cache[name]
}

func (e *bindConstructor) setBinding(name string, bs []uint8) {
	e.mux.Lock()
	defer e.mux.Unlock()
	if e.cache == nil {
		e.cache = make(map[string][]uint8)
	}
	e.cache[name] = bs
}
