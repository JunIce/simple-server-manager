# 模块：运行时管理（internal/runtime）

## 职责

管理主机上安装的**运行时环境**（以 JDK 为典型），提供运行时的注册、发现、解析与校验，使不同服务可绑定不同版本的运行时启动。

## 核心概念

| 概念 | 说明 |
|------|------|
| Runtime | 一种运行时环境实例，如某个具体版本的 JDK |
| RuntimeKind | 运行时类型：`jdk`（后续可扩展 `node`、`python` 等） |
| RuntimeManager | 运行时注册表，负责发现、注册、解析 |

## 核心类型

### Runtime

```go
type Runtime struct {
    ID      string `json:"id"`      // 唯一标识，如 "jdk-17"、"jdk-1.8"
    Name    string `json:"name"`    // 显示名，如 "OpenJDK 17"
    Kind    string `json:"kind"`    // 运行时类型，默认 "jdk"
    Version string `json:"version"` // 版本号，如 "17.0.10"
    Home    string `json:"home"`    // 安装根目录，如 C:\Program Files\Java\jdk-17
    Default bool   `json:"default"` // 是否默认运行时
}
```

### RuntimeManager

```go
type RuntimeManager struct {
    mu        sync.RWMutex
    runtimes  map[string]*Runtime
    resolver  *Resolver   // 将运行时解析为可执行文件路径
    discover  *Discoverer // 自动发现已安装运行时
}
```

## 运行时发现（Discoverer）

自动扫描主机上已安装的 JDK，减少手动注册成本。

```go
type Discoverer struct{}

func (d *Discoverer) Discover(kind string) ([]*Runtime, error) {
    switch kind {
    case "jdk":
        return d.discoverJDK()
    default:
        return nil, ErrUnknownKind
    }
}

func (d *Discoverer) discoverJDK() ([]*Runtime, error) {
    var found []*Runtime
    // 1. JAVA_HOME 环境变量
    // 2. 常见安装目录扫描：
    //    C:\Program Files\Java\*
    //    C:\Program Files\Eclipse Adoptium\*
    //    C:\Program Files\Microsoft\*
    // 3. 注册表：HKLM\SOFTWARE\JavaSoft\JDK\*
    // 4. 执行 java -version 校验并解析版本号
    return found, nil
}
```

## 运行时解析（Resolver）

将运行时解析为具体可执行文件路径，供 handler 构建命令。

```go
type Resolver struct{}

func (r *Resolver) Executable(rt *Runtime) (string, error) {
    switch rt.Kind {
    case "jdk":
        // Windows: <home>\bin\java.exe
        return filepath.Join(rt.Home, "bin", "java.exe"), nil
    default:
        return "", ErrUnknownKind
    }
}

// 校验运行时可用：检查可执行文件存在且可执行
func (r *Resolver) Validate(rt *Runtime) error {
    exe, err := r.Executable(rt)
    if err != nil { return err }
    if _, err := os.Stat(exe); err != nil {
        return ErrRuntimeInvalid
    }
    return nil
}
```

## 管理操作

```go
func (m *RuntimeManager) Register(rt *Runtime) error {
    if err := m.resolver.Validate(rt); err != nil {
        return err
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    if rt.Default {
        m.clearDefault(rt.Kind)  // 同类型仅保留一个默认
    }
    m.runtimes[rt.ID] = rt
    return m.persist()          // 持久化到 JSON
}

func (m *RuntimeManager) Resolve(ref string, kind string) (*Runtime, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    // 1. ref 为空 → 返回该类型默认运行时
    // 2. ref 精确匹配 ID → 返回
    // 3. ref 匹配 Version 前缀 → 返回
    // 4. 找不到 → ErrRuntimeNotFound
    return nil, ErrRuntimeNotFound
}
```

## 与服务绑定

服务通过 `JavaConfig.JDK`（或通用 `RuntimeRef`）引用运行时：

```json
{
  "id": "legacy-app",
  "type": "java",
  "java": {
    "jdk": "jdk-1.8",
    "jarFile": "legacy.jar"
  }
}
```

启动时由 `JavaHandler` 调用 `RuntimeManager.Resolve` 解析 JDK，再通过 `Resolver.Executable` 得到 `java.exe` 绝对路径构建命令。

## 持久化

运行时注册表随 `Config` 一起持久化到 JSON 文件：

```json
{
  "runtimes": {
    "jdk-17": {
      "id": "jdk-17",
      "name": "OpenJDK 17",
      "kind": "jdk",
      "version": "17.0.10",
      "home": "C:\\Program Files\\Java\\jdk-17",
      "default": true
    },
    "jdk-1.8": {
      "id": "jdk-1.8",
      "name": "Oracle JDK 8",
      "kind": "jdk",
      "version": "1.8.0_401",
      "home": "C:\\Program Files\\Java\\jdk1.8.0_401"
    }
  }
}
```

## 设计要点

- 运行时注册表独立于服务配置，可被多个服务复用。
- 自动发现 + 手动注册双通道，兼顾易用与精确。
- `Resolve` 支持「默认 / 精确 ID / 版本前缀」三级解析，服务可灵活引用。
- 扩展新运行时类型（node/python 等）只需在 `Resolver`/`Discoverer` 增加分支。
