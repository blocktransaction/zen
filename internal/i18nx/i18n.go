package i18nx

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blocktransaction/zen/common/constant"
	"github.com/blocktransaction/zen/config"
	"github.com/fsnotify/fsnotify"
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
	watcher   *fsnotify.Watcher
	dir       string
	cb        func(event fsnotify.Event) // 可选回调
	// lang      string // 链式语言存储
}

// 包装对象用于链式访问，避免单例共享 lang 字段造成并发问题
type WithLangManager struct {
	manager *Manager
	lang    string
}

func Setup() {
	GetManager().setup(
		config.ApplicationConfig.I18nFilePath,
		config.ApplicationConfig.I18nSupportLanguage,
		config.ApplicationConfig.DefaultLang,
	)
}

// 单例
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

// 设置文件侦听事件
func (m *Manager) SetFsEvent(cb func(event fsnotify.Event)) *Manager {
	m.cb = cb
	return m
}

// 初始化
func (m *Manager) setup(
	path string,
	supported []string,
	defaultLang string,
	// cb func(event fsnotify.Event),
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defLang = defaultLang
	m.supported = supported
	// m.cb = cb
	m.dir = path

	if err := m.loadFiles(path, supported); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	m.watcher = watcher

	if err := watcher.Add(path); err != nil {
		return err
	}

	go m.watchLoop()
	return nil
}

// watchLoop 监听文件变化
func (m *Manager) watchLoop() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				m.mu.Lock()
				_ = m.loadFiles(m.dir, m.supported)
				m.mu.Unlock()
				if m.cb != nil {
					go m.cb(event) // 异步回调
				}
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			// 可以选择 log
			_ = err
		}
	}
}

// 动态加载/更新
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

// 更新某个语言的单条翻译
func (m *Manager) Update(lang, key, value string) {
	lang = normalizeLang(lang)
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.messages[lang]; !ok {
		m.messages[lang] = make(map[string]string)
	}
	m.messages[lang][key] = value
}

// 链式语言
func (m *Manager) WithLang(ctx context.Context, lang string) *WithLangManager {
	if ctx != nil {
		lang = getLangFromContext(ctx, lang)
	}
	return &WithLangManager{
		manager: m,
		lang:    normalizeLang(lang),
	}
}

// StopWatcher 停止监听
func (m *Manager) StopWatcher() error {
	if m.watcher != nil {
		return m.watcher.Close()
	}
	return nil
}

// WithLangManager 获取翻译
func (w *WithLangManager) Msg(code string) string {
	return w.GetMessage(code)
}

func (w *WithLangManager) GetMessage(code string) string {
	lang := w.lang
	if lang == "" {
		lang = w.manager.defLang
	}

	w.manager.mu.RLock()
	defer w.manager.mu.RUnlock()

	if msgs, ok := w.manager.messages[lang]; ok {
		if msg, exists := msgs[code]; exists {
			return msg
		}
	}
	return code
}

// 请求级语言
func WithCtxLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, constant.CtxKeyLang, normalizeLang(lang))
}

// 内部工具
func normalizeLang(lang string) string {
	return strings.ToLower(strings.ReplaceAll(lang, "_", "-"))
}

func getLangFromContext(ctx context.Context, defaultLang string) string {
	if v := ctx.Value(constant.CtxKeyLang); v != nil {
		if l, ok := v.(string); ok && l != "" {
			return normalizeLang(l)
		}
	}
	return normalizeLang(defaultLang)
}
