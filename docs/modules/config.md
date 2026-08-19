# 模块：配置管理（internal/config）

## 职责

加载、校验、持久化系统配置与服务配置到 **JSON 文件**（本地或用户目录），支持运行期热更新，**不依赖任何数据库**。

## 配置结构

```go
type Config struct {
	Server   ServerConfig            `json:"server"`
	Services map[string]ServiceConfig `json:"services"`
	JDKs     map[string]JDK          `json:"jdks"`   // JDK 版本配置（统一配置页管理）
	Log      LogConfig               `json:"log"`
	Monitor  MonitorConfig           `json:"monitor"`
}

type ServerConfig struct {
    Port     int    `json:"port"`
    Host     string `json:"host"`
    Auth     AuthConfig `json:"auth"`
    DataDir  string `json:"dataDir"`
}

type AuthConfig struct {
    Enabled bool   `json:"enabled"`
    Token   string `json:"token"`
}

type LogConfig struct {
    OutputDir   string `json:"outputDir"`
    MaxSizeMB   int64  `json:"maxSizeMB"`   // 滚动大小
    MaxFiles    int    `json:"maxFiles"`    // 保留文件数
    BufferSize  int    `json:"bufferSize"`
}

type MonitorConfig struct {
    IntervalSeconds int `json:"intervalSeconds"`
}
```

## 存储位置解析

```go
func resolvePath(explicit string) (string, error) {
    if explicit != "" {
        return explicit, nil
    }
    if dir, err := os.UserConfigDir(); err == nil {
        // Windows: %LOCALAPPDATA%\service-manager\config.json
        return filepath.Join(dir, "service-manager", "config.json"), nil
    }
    // 兜底：用户主目录
    if home, err := os.UserHomeDir(); err == nil {
        return filepath.Join(home, ".service-manager", "config.json"), nil
    }
    return "config.json", nil  // 最终兜底：当前目录
}
```

## 加载流程

```go
func Load(explicit string) (*Config, error) {
    path, _ := resolvePath(explicit)
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return Default(), nil  // 不存在则用默认配置
        }
        return nil, err
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

## 校验规则

- `Server.Port` 范围 1–65535。
- 服务 `ID` 非空且唯一。
- 服务 `Command` 非空。
- 服务 `Type` 必须已注册 handler。
- 监控周期 > 0。

## 持久化

```go
func (c *Config) Save(path string) error {
    c.mu.RLock()
    defer c.mu.RUnlock()
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    return atomicWrite(path, data)  // 临时文件 + rename 原子写
}
```

- 使用临时文件 + `os.Rename` 原子写，避免中途崩溃损坏配置。

## 热更新

- 服务配置变更通过 `core.UpdateService` 触发，`ServiceManager.mu` 保证并发一致性。
- 全局配置（如监控周期）变更后重建相关 ticker。
- 使用 `sync.RWMutex` 保护配置读。

## JDK 版本管理

系统上不同版本的 JDK 由统一配置页集中管理，数据落在 `Config.JDKs`：

- **模型**：`JDK{ ID, Name, Version, Home, Default }`，`Home` 为安装根目录，解析为 `<home>\bin\java.exe`。
- **自动发现**：`DiscoverJDKs()` 扫描 `JAVA_HOME` 与常见安装目录，执行 `java -version` 校验并解析版本号。
- **三级解析**：`ResolveJDK(ref)` —— `ref` 为空取默认或第一个；精确匹配 ID；版本或 ID 前缀匹配。
- **写回**：`UpdateJDK` / `DeleteJDK` / `UpdateSystem`（整体原子保存），同类型仅保留一个 `Default`。
- **绑定**：Java 服务通过 `ServiceConfig.Java.JDK` 引用 JDK ID，启动时由 `JavaHandler` 解析出 `java.exe` 并注入 `JAVA_HOME`。

## 配置文件示例

```json
{
  "server": { "port": 8080, "host": "0.0.0.0", "dataDir": "./data" },
  "jdks": {
    "jdk-17": {
      "id": "jdk-17",
      "name": "OpenJDK 17",
      "version": "17.0.10",
      "home": "C:\\Program Files\\Java\\jdk-17",
      "default": true
    },
    "jdk-1.8": {
      "id": "jdk-1.8",
      "name": "Oracle JDK 8",
      "version": "1.8.0_401",
      "home": "C:\\Program Files\\Java\\jdk1.8.0_401"
    }
  },
  "services": {
    "my-java-app": {
      "id": "my-java-app",
      "name": "My Java Application",
      "type": "java",
      "java": { "jdk": "jdk-17", "jarFile": "app.jar" },
      "environments": ["dev", "test", "prod"],
      "autoRestart": true
    },
    "legacy-app": {
      "id": "legacy-app",
      "name": "Legacy Java App",
      "type": "java",
      "java": { "jdk": "jdk-1.8", "jarFile": "legacy.jar" }
    }
  },
  "log": { "outputDir": "./data/logs", "maxSizeMB": 50, "maxFiles": 10 },
  "monitor": { "intervalSeconds": 5 }
}
```
