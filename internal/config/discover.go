package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var jdkVersionRe = regexp.MustCompile(`version "([^"]+)"`)

// DiscoverJDKs 自动扫描主机上已安装的 JDK 并解析版本号。
// 探测顺序：JAVA_HOME 环境变量 → 常见安装目录。
func DiscoverJDKs() ([]JDK, error) {
	seen := map[string]bool{}
	var found []JDK

	// 1. JAVA_HOME 环境变量
	if home := os.Getenv("JAVA_HOME"); home != "" {
		if j, ok := buildJDK(home); ok {
			found = append(found, j)
			seen[strings.ToLower(home)] = true
		}
	}

	// 2. 常见安装目录
	dirs := []string{
		`C:\Program Files\Java`,
		`C:\Program Files\Eclipse Adoptium`,
		`C:\Program Files\Microsoft`,
		`C:\Program Files\Zulu`,
		`C:\Program Files\Amazon Corretto`,
		`C:\Program Files (x86)\Java`,
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			home := filepath.Join(dir, e.Name())
			if seen[strings.ToLower(home)] {
				continue
			}
			if j, ok := buildJDK(home); ok {
				found = append(found, j)
				seen[strings.ToLower(home)] = true
			}
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].ID < found[j].ID })
	return found, nil
}

func buildJDK(home string) (JDK, bool) {
	exe := filepath.Join(home, "bin", "java.exe")
	if _, err := os.Stat(exe); err != nil {
		return JDK{}, false
	}
	version := detectJDKVersion(exe)
	id := "jdk-" + sanitizeJDKID(version)
	if id == "jdk-" {
		id = "jdk-" + sanitizeJDKID(filepath.Base(home))
	}
	return JDK{
		ID:      id,
		Name:    "JDK " + version,
		Version: version,
		Home:    home,
	}, true
}

func detectJDKVersion(javaExe string) string {
	cmd := exec.Command(javaExe, "-version")
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return "unknown"
	}
	// java -version 输出到 stderr
	m := jdkVersionRe.FindStringSubmatch(string(out))
	if len(m) >= 2 {
		return m[1]
	}
	return "unknown"
}

func sanitizeJDKID(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
