// Package web 内嵌前端静态资源与核心插件清单，使单个二进制即可独立部署。
//
// 目录布局（与 public/ 保持一致，便于「磁盘优先、内嵌兜底」）：
//
//	dist/assets/admin/**          管理端构建产物（admin-src → public/assets/admin）
//	dist/theme/<主题名>/**         用户端主题（theme/Xboard）
//	dist/plugins-core/<目录>/**    核心插件的 config.json / README.md
//
// 内容由 scripts/sync-web.sh 从仓库根目录同步；发布脚本构建前会自动执行。
package web

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// FS 返回以 dist 为根的只读文件系统。
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return embedded
	}
	return sub
}

func clean(name string) string { return strings.TrimPrefix(path.Clean("/"+name), "/") }

// ReadFile 读取内嵌文件；name 为相对 dist 的路径，自动拒绝越界。
func ReadFile(name string) ([]byte, error) { return fs.ReadFile(FS(), clean(name)) }

// Exists 判断内嵌文件（非目录）是否存在。
func Exists(name string) bool {
	st, err := fs.Stat(FS(), clean(name))
	return err == nil && !st.IsDir()
}

// DirNames 列出内嵌目录下的子项名称（已排序）；目录不存在时返回 nil。
func DirNames(dir string, onlyDirs bool) []string {
	entries, err := fs.ReadDir(FS(), clean(dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Name() == ".keep" || (onlyDirs && !e.IsDir()) {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}
