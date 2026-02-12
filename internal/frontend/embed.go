package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var frontendFS embed.FS

// FS 返回前端静态文件系统
func FS() http.FileSystem {
	sub, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		// 如果出错，返回空文件系统
		return &nopFS{}
	}
	return http.FS(sub)
}

// HasFiles 检查是否存在前端构建文件
func HasFiles() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}

// nopFS 空文件系统，用于前端构建不存在时
type nopFS struct{}

func (f *nopFS) Open(name string) (http.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
