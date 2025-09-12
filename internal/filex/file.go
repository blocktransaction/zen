package filex

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles 列出目录下符合条件的文件
// path: 目录路径
// pattern: 通配符 (例如 "*.json", "config*.yaml")
// ignoreCase: 是否忽略大小写
// recursive: 是否递归子目录
func ListFiles(path, pattern string, ignoreCase, recursive bool) ([]string, error) {
	// 转绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// 确认目录存在
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("not a directory: " + absPath)
	}

	// 如果忽略大小写，就把 pattern 转小写
	patternLower := strings.ToLower(pattern)

	var files []string
	if recursive {
		// Walk 递归
		err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				name := d.Name()
				matchName := name
				matchPattern := pattern
				if ignoreCase {
					matchName = strings.ToLower(name)
					matchPattern = patternLower
				}
				matched, err := filepath.Match(matchPattern, matchName)
				if err != nil {
					return err
				}
				if matched {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		// 仅当前目录
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, err
		}
		for _, f := range entries {
			if !f.IsDir() {
				name := f.Name()
				matchName := name
				matchPattern := pattern
				if ignoreCase {
					matchName = strings.ToLower(name)
					matchPattern = patternLower
				}
				matched, err := filepath.Match(matchPattern, matchName)
				if err != nil {
					return nil, err
				}
				if matched {
					files = append(files, filepath.Join(absPath, name))
				}
			}
		}
	}

	return files, nil
}
