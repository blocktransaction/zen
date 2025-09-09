package i18n

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blocktransaction/zen/config"
)

// 特性总结
// 全局单例，只加载一次翻译。
// 线程安全，支持并发读取和写入。
// Fluent 链式 API，Service 层写法简洁：.WithLang(ctx, lang).Msg("code")。
// 多级 fallback：支持 zh-CN → zh → en。
// 请求级语言：从 context 获取语言，Web 并发安全。
// 动态加载 JSON 文件：LoadFiles(path) 可在运行时更新所有翻译。
// 单条 key 动态更新：Update(lang, key, value) 可在线更新某条翻译，无需重启。
// 安全 fallback：找不到翻译返回原始 code，不会 panic。

// ----------------- Manager -----------------

var (
	once    sync.Once
	manager *Manager
)

type Manager struct {
	mu        sync.RWMutex
	messages  map[string]map[string]string
	defLang   string
	supported []string
	lang      string // 链式语言存储
}
type ctxKey string

const langKey ctxKey = "lang"

func Setup() {
	GetManager().Setup(
		config.ApplicationConfig.I18nFilePath,
		config.ApplicationConfig.I18nSupportLanguage,
		config.ApplicationConfig.DefaultLang)
}

// ----------------- 单例 -----------------
func GetManager() *Manager {
	once.Do(func() {
		manager = &Manager{
			messages:  make(map[string]map[string]string),
			defLang:   "en",
			supported: []string{"en"},
		}
	})
	return manager
}

// ----------------- 初始化 -----------------

func (m *Manager) Setup(path string, supported []string, defaultLang string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defLang = defaultLang
	m.supported = supported
	return m.loadFiles(path, supported)
}

// ----------------- 动态加载/更新 -----------------

func (m *Manager) LoadFiles(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadFiles(path, m.supported)
}

func (m *Manager) loadFiles(path string, supported []string) error {
	files, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		return err
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var content map[string]string
		if err := json.Unmarshal(data, &content); err != nil {
			continue
		}
		for _, lang := range supported {
			if strings.Contains(filepath.Base(f), lang) {
				if _, ok := m.messages[lang]; !ok {
					m.messages[lang] = make(map[string]string)
				}
				for k, v := range content {
					m.messages[lang][k] = v
				}
			}
		}
	}
	return nil
}

// ----------------- 单条 key 更新 -----------------

// Update 更新某个语言的单条翻译
func (m *Manager) Update(lang, key, value string) {
	lang = normalizeLang(lang)
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.messages[lang]; !ok {
		m.messages[lang] = make(map[string]string)
	}
	m.messages[lang][key] = value
}

// ----------------- 链式语言 -----------------

func (m *Manager) WithLang(ctx context.Context, lang string) *Manager {
	if ctx != nil {
		lang = getLangFromContext(ctx, lang)
	}
	m.lang = normalizeLang(lang)
	return m
}

func getLangFromContext(ctx context.Context, defaultLang string) string {
	if v := ctx.Value(langKey); v != nil {
		if l, ok := v.(string); ok && l != "" {
			return normalizeLang(l)
		}
	}
	return normalizeLang(defaultLang)
}

// ----------------- Fluent API -----------------

func (m *Manager) Msg(code string) string {
	return m.GetMessage(code)
}

// ----------------- 获取翻译 -----------------

func (m *Manager) GetMessage(code string) string {
	lang := m.lang
	if lang == "" {
		lang = m.defLang
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, l := range fallbackChain(lang, m.defLang) {
		if msgs, ok := m.messages[l]; ok {
			if msg, exists := msgs[code]; exists {
				return msg
			}
		}
	}
	return code
}

// ----------------- 请求级语言 -----------------

func WithCtxLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, langKey, normalizeLang(lang))
}

// ----------------- 内部工具 -----------------

func fallbackChain(lang, defLang string) []string {
	var chain []string
	lang = normalizeLang(lang)
	if strings.Contains(lang, "-") {
		parts := strings.Split(lang, "-")
		chain = append(chain, lang, parts[0])
	} else {
		chain = append(chain, lang)
	}

	if defLang != lang {
		chain = append(chain, normalizeLang(defLang))
	}

	return chain
}

func normalizeLang(lang string) string {
	return strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
}
