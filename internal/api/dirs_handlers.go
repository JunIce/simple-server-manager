package api

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// dirEntry 目录浏览条目。
type dirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// listDirs GET /api/dirs?path=xxx 列出指定路径下的子目录。
// path 为空时，Windows 返回磁盘盘符，其他平台返回根目录 "/"。
func (s *Server) listDirs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	var entries []dirEntry

	// path 为空 → 返回盘符（Windows）或根目录
	if path == "" {
		if runtime.GOOS == "windows" {
			for _, drive := range drives() {
				entries = append(entries, dirEntry{Name: drive, Path: drive, IsDir: true})
			}
		} else {
			entries = append(entries, dirEntry{Name: "/", Path: "/", IsDir: true})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
		writeJSON(w, entries)
		return
	}

	fis, err := os.ReadDir(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	for _, fi := range fis {
		if !fi.IsDir() {
			continue // 只返回目录
		}
		child := filepath.Join(path, fi.Name())
		entries = append(entries, dirEntry{Name: fi.Name(), Path: child, IsDir: true})
	}
	sort.Slice(entries, func(i, j int) bool { return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name) })
	writeJSON(w, entries)
}

// drives 返回 Windows 逻辑磁盘盘符（如 C:\、D:\）。
func drives() []string {
	bits, err := drivesBitmask()
	if err != nil {
		return nil
	}
	var out []string
	for i := 0; i < 26; i++ {
		if bits&(1<<uint(i)) != 0 {
			out = append(out, string(rune('A'+i))+":\\")
		}
	}
	return out
}
