// Package webui 嵌入 Web 前端静态资源。
package webui

import "embed"

// FS Web 前端静态资源（由 web/ 通过 vite build 输出到本目录）。
//
//go:embed all:static
var FS embed.FS
