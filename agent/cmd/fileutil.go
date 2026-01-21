package cmd

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func isInManagedScope(filePath string, scopes map[string]struct{}) bool {
	abs, _ := filepath.Abs(filePath)
	for scope := range scopes {
		info, err := os.Stat(scope)
		if err != nil {
			continue
		}
		if info.IsDir() {
			if strings.HasPrefix(abs, scope+string(os.PathSeparator)) || abs == scope {
				return true
			}
		} else {
			if abs == scope {
				return true
			}
		}
	}
	return false
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func cleanupExtraFiles(scopes map[string]struct{}, keep map[string]struct{}) error {
	for scope := range scopes {
		info, err := os.Stat(scope)
		if err != nil {
			continue
		}
		if info.IsDir() {
			err = filepath.Walk(scope, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				abs := filepath.Clean(path)
				if _, ok := keep[abs]; !ok {
					if rmErr := os.Remove(abs); rmErr != nil {
						log.Printf("⚠️ 删除文件失败 %s: %v", abs, rmErr)
					} else {
						log.Printf("🧹 删除多余文件: %s", abs)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			abs := filepath.Clean(scope)
			if _, ok := keep[abs]; !ok {
				if err := os.WriteFile(abs, []byte{}, info.Mode()); err != nil {
					log.Printf("⚠️ 清空文件失败 %s: %v", abs, err)
				} else {
					log.Printf("🧹 清空文件: %s", abs)
				}
			}
		}
	}
	return nil
}
