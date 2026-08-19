package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// fileEntry 文件/目录条目。
type fileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"` // 相对根目录的路径，使用 / 分隔
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

// securePath 将相对路径安全解析到根目录下，防止路径穿越。
func (s *Server) securePath(rel string) (abs string, err error) {
	root := s.cfg.GetServer().FileRoot
	if root == "" {
		return "", &os.PathError{Op: "resolve", Path: rel, Err: os.ErrNotExist}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel = strings.ReplaceAll(rel, "/", string(filepath.Separator))
	rel = filepath.Clean(rel)
	rel = strings.TrimLeft(rel, string(filepath.Separator))
	abs = filepath.Join(absRoot, rel)
	// 校验仍在根目录内
	checkRel, err := filepath.Rel(absRoot, abs)
	if err != nil {
		return "", err
	}
	if checkRel == ".." || strings.HasPrefix(checkRel, ".."+string(filepath.Separator)) || filepath.IsAbs(checkRel) {
		return "", &os.PathError{Op: "check", Path: rel, Err: os.ErrPermission}
	}
	return abs, nil
}

// listFiles GET /api/files?path=rel 列出根目录下某子目录的内容。
func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.securePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	fis, err := os.ReadDir(abs)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	entries := make([]fileEntry, 0, len(fis))
	for _, fi := range fis {
		info, _ := fi.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}
		entries = append(entries, fileEntry{
			Name:  fi.Name(),
			Path:  toSlash(filepath.Join(rel, fi.Name())),
			IsDir: fi.IsDir(),
			Size:  size,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir // 目录在前
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	writeJSON(w, entries)
}

// readFileContent GET /api/files/content?path=rel 读取文本文件内容。
func (s *Server) readFileContent(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.securePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	writeJSON(w, string(b))
}

// writeFileContent PUT /api/files/content?path=rel 保存文件内容（body 为 JSON 字符串）。
func (s *Server) writeFileContent(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.securePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	var content string
	if err := json.NewDecoder(r.Body).Decode(&content); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid body: "+err.Error())
		return
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"path": rel, "saved": true})
}

// downloadFile GET /api/files/download?path=rel 下载文件。
func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	abs, err := s.securePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, 1002, "cannot download a directory")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+urlQueryEscape(filepath.Base(abs)))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, abs)
}

// uploadFile POST /api/files/upload?path=dir 上传文件到指定目录。
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	dirRel := r.URL.Query().Get("path")
	dirAbs, err := s.securePath(dirRel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	defer file.Close()

	name := filepath.Base(header.Filename)
	destAbs, err := s.securePath(filepath.Join(dirRel, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	dst, err := os.Create(destAbs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	_ = dirAbs
	writeJSON(w, map[string]interface{}{"saved": name})
}

// renameFile POST /api/files/rename body {path, newName} 重命名文件/目录。
func (s *Server) renameFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid body: "+err.Error())
		return
	}
	abs, err := s.securePath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	newName := filepath.Base(strings.ReplaceAll(req.NewName, "/", string(filepath.Separator)))
	if newName == "" || newName == "." || newName == ".." {
		writeError(w, http.StatusBadRequest, 1002, "invalid name")
		return
	}
	newAbs, err := s.securePath(filepath.Join(filepath.Dir(req.Path), newName))
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if err := os.Rename(abs, newAbs); err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"renamed": newName})
}

// deleteFile DELETE /api/files/delete?path=rel 删除文件/目录（递归）。
func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		writeError(w, http.StatusBadRequest, 1002, "path is required")
		return
	}
	abs, err := s.securePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if rel == "." {
		writeError(w, http.StatusBadRequest, 1002, "cannot delete root")
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"deleted": rel})
}

// mkdirFile POST /api/files/mkdir body {path, name} 新建文件夹。
func (s *Server) mkdirFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid body: "+err.Error())
		return
	}
	newName := filepath.Base(strings.ReplaceAll(req.Name, "/", string(filepath.Separator)))
	if newName == "" || newName == "." || newName == ".." {
		writeError(w, http.StatusBadRequest, 1002, "invalid name")
		return
	}
	newAbs, err := s.securePath(filepath.Join(req.Path, newName))
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if err := os.MkdirAll(newAbs, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, 500, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"created": req.Path + "/" + newName})
}

// copyFile POST /api/files/copy body {source, dest} 复制文件/目录到目标位置。
func (s *Server) copyFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
		Dest   string `json:"dest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 1002, "invalid body: "+err.Error())
		return
	}
	srcAbs, err := s.securePath(req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	destAbs, err := s.securePath(req.Dest)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	info, err := os.Stat(srcAbs)
	if err != nil {
		writeError(w, http.StatusBadRequest, 1002, err.Error())
		return
	}
	if info.IsDir() {
		if err := copyDir(srcAbs, destAbs); err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return
		}
	} else {
		if err := copyFileTo(srcAbs, destAbs); err != nil {
			writeError(w, http.StatusInternalServerError, 500, err.Error())
			return
		}
	}
	writeJSON(w, map[string]interface{}{"copied": req.Source, "to": req.Dest})
}

func copyFileTo(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFileTo(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

func urlQueryEscape(s string) string {
	// 用 RFC5987 的百分号编码兜底，避免文件名中文在 header 出错
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-' || r == '_' || r == '.' || r == '~':
			b.WriteRune(r)
		default:
			fmt.Fprintf(&b, "%%%02X", r)
		}
	}
	return b.String()
}

