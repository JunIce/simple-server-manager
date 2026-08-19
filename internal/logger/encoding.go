package logger

import (
	"bufio"
	"bytes"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// textDecoder 子进程输出编码解码器：自动检测 UTF-8 / GBK。
type textDecoder struct{}

// newTextDecoder 创建编码解码器。
func newTextDecoder() *textDecoder { return &textDecoder{} }

// Reader 包装原始 reader，统一输出 UTF-8。
// 中文 Windows 上子进程（如 ping）常输出 GBK，需转为 UTF-8。
// 采用增量检测，避免 Peek 大块阻塞导致实时流停滞。
func (d *textDecoder) Reader(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	return &detectingReader{br: br}
}

// detectingReader 增量编码检测 reader。
type detectingReader struct {
	br      *bufio.Reader
	buf     []byte
	decided bool
	gbk     io.Reader
	utf8r   io.Reader
}

func (d *detectingReader) Read(p []byte) (int, error) {
	if !d.decided {
		d.buf, d.decided = d.detect()
		if d.decided {
			if d.gbk != nil {
				return d.gbk.Read(p)
			}
			d.utf8r = io.MultiReader(bytes.NewReader(d.buf), d.br)
		}
	}
	if d.gbk != nil {
		return d.gbk.Read(p)
	}
	if d.utf8r != nil {
		return d.utf8r.Read(p)
	}
	return d.br.Read(p)
}

// detect 逐字节收集并判定编码。判定 GBK 后立即返回；收集满窗口仍全部合法则按 UTF-8。
func (d *detectingReader) detect() ([]byte, bool) {
	const window = 1024
	for len(d.buf) < window {
		b, err := d.br.ReadByte()
		if err != nil {
			break
		}
		d.buf = append(d.buf, b)
		if definitelyInvalidUTF8(d.buf) {
			// 明确非法字节，判定为 GBK，从当前位置开始用 GBK 解码
			src := io.MultiReader(bytes.NewReader(d.buf), d.br)
			d.gbk = transform.NewReader(src, simplifiedchinese.GBK.NewDecoder())
			return d.buf, true
		}
	}
	// 窗口内无非法字节，视为 UTF-8（或纯 ASCII）
	return d.buf, true
}

// definitelyInvalidUTF8 判断 buf 是否包含明确非法的 UTF-8 字节序列。
// 尾部不完整的 rune 视为「待更多数据」，不据此判定。
func definitelyInvalidUTF8(buf []byte) bool {
	i := 0
	for i < len(buf) {
		if !utf8.FullRune(buf[i:]) {
			return false // 尾部为不完整 rune，等待更多数据
		}
		r, size := utf8.DecodeRune(buf[i:])
		if r == utf8.RuneError && size == 1 {
			return true // 明确非法字节
		}
		i += size
	}
	return false
}
