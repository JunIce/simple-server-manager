# 模块：通用工具（pkg/utils）

## 职责

提供无业务语义的通用工具函数，供各模块复用。

## 功能清单

| 函数 | 说明 |
|------|------|
| `BackoffDelay(policy, retries)` | 计算退避延迟（fixed / exponential + 抖动） |
| `AtomicWrite(path, data)` | 临时文件 + rename 原子写 |
| `ParseLogLevel(line)` | 从日志行提取级别 |
| `ContainsAll(s, subs)` | 判断包含所有子串 |
| `TruncateString(s, n)` | 字符串截断 |
| `MapToEnv(m)` | `map[string]string` 转 `KEY=value` 切片 |

## 退避延迟

```go
func BackoffDelay(policy RestartPolicy, retries int) time.Duration {
    var d time.Duration
    switch policy.BackoffType {
    case "exponential":
        d = policy.BackoffDelay << uint(retries)
        if d > policy.MaxBackoff || d <= 0 {
            d = policy.MaxBackoff
        }
    default: // fixed
        d = policy.BackoffDelay
    }
    // ±10% 抖动
    jitter := time.Duration(rand.Int63n(int64(d) / 5)) - d/10
    return d + jitter
}
```

## 原子写

```go
func AtomicWrite(path string, data []byte) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil { return err }
    defer os.Remove(tmp.Name())

    if _, err := tmp.Write(data); err != nil { return err }
    if err := tmp.Sync(); err != nil { return err }
    if err := tmp.Close(); err != nil { return err }
    return os.Rename(tmp.Name(), path)
}
```

## 环境变量转换

```go
func MapToEnv(m map[string]string) []string {
    env := make([]string, 0, len(m))
    for k, v := range m {
        env = append(env, k+"="+v)
    }
    sort.Strings(env)  // 稳定顺序便于测试与调试
    return env
}
```
