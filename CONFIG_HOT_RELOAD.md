# 配置热重载功能 (Configuration Hot Reload)

## 概述

`zzz` 工具现在支持配置热重载功能，允许在运行时动态更新配置文件，无需重启应用程序。当配置文件发生变化时，系统会自动检测并重新加载配置。

## 功能特性

### 1. 自动检测配置变化
- 使用 `fsnotify` 监控配置文件变化
- 支持实时检测 `.zzz.yaml` 文件的修改
- 自动触发配置重载流程

### 2. 智能配置更新
- 原子性配置更新，确保数据一致性
- 支持配置验证，防止无效配置
- 详细的变更日志记录

### 3. 回调机制
- 支持注册配置变更回调函数
- 可扩展的事件处理机制
- 错误处理和恢复机制

## 技术实现

### 核心组件

#### ConfigReloader 结构
```go
type ConfigReloader struct {
    configPath   string
    watcher      *fsnotify.Watcher
    callbacks    []ConfigCallback
    isRunning    bool
    lastModified time.Time
    mu           sync.RWMutex
    stats        map[string]interface{}
}
```

#### 主要方法
- `NewConfigReloader(configPath string)`: 创建配置重载器
- `Start()`: 启动文件监控
- `Stop()`: 停止监控
- `AddCallback(callback ConfigCallback)`: 添加回调函数
- `ForceReload()`: 强制重载配置
- `GetStats()`: 获取统计信息

### 集成点

#### 1. 初始化 (internal/cmd/run.go)
```go
// 在 init() 函数中初始化配置热重载
configReloader, err = hotreload.NewConfigReloader(file)
if err != nil {
    logger.Log.Warnf("Failed to initialize config hot reload: %s", err)
} else {
    configReloader.AddCallback(onConfigReload)
    configReloader.Start()
}
```

#### 2. 回调处理 (internal/cmd/run.go)
```go
// 配置变更回调函数
func onConfigReload(newConfigData interface{}) error {
    // 解析新配置
    // 原子性更新
    // 记录变更日志
    return nil
}
```

## 使用方法

### 1. 查看热重载状态
```bash
# 查看系统状态，包含热重载信息
./zzz status

# 输出示例：
=== Configuration Hot Reload ===
Running: true
Config Path: /path/to/.zzz.yaml
Last Modified: 2025-07-26T15:54:47+08:00
Callbacks: 1
```

### 2. 强制重载配置
```bash
# 手动触发配置重载
./zzz optimize --reload-config
```

### 3. 自动热重载
当运行 `./zzz run` 时，系统会自动监控配置文件变化：

```bash
# 启动应用
./zzz run

# 在另一个终端修改 .zzz.yaml 文件
# 系统会自动检测并重载配置
```

## 监控的配置项

系统会监控以下配置项的变化并记录日志：

- `frequency`: 监控频率变化
- `lang`: 编程语言变化
- `ext`: 文件扩展名变化
- `dirfilter`: 目录过滤规则变化
- `action`: 构建动作变化

## 日志示例

### 配置变更检测
```
2025/07/26 15:54:47 INFO Configuration file changed, reloading...
2025/07/26 15:54:47 INFO Frequency changed from 5 to 2 seconds
2025/07/26 15:54:47 SUCCESS Configuration hot reloaded successfully
```

### 强制重载
```
2025/07/26 15:54:20 INFO Forcing configuration reload...
2025/07/26 15:54:20 SUCCESS Configuration reloaded successfully
```

## 错误处理

### 配置文件格式错误
- 系统会保留原有配置
- 记录错误日志
- 不会中断应用运行

### 文件监控失败
- 降级到传统的轮询检查
- 记录警告日志
- 保持基本功能可用

## 性能优化

### 1. 防抖机制
- 避免频繁的配置重载
- 合并短时间内的多次变更

### 2. 内存管理
- 使用对象池管理配置对象
- 及时释放不再使用的资源

### 3. 并发安全
- 使用读写锁保护配置数据
- 原子性操作确保数据一致性

## 扩展性

### 添加新的配置回调
```go
// 注册自定义回调
configReloader.AddCallback(func(newConfig interface{}) error {
    // 处理配置变更
    return nil
})
```

### 监控其他配置文件
```go
// 创建额外的配置重载器
additionalReloader := hotreload.NewConfigReloader("/path/to/other/config.yaml")
```

## 最佳实践

1. **配置验证**: 在回调函数中验证新配置的有效性
2. **渐进式更新**: 对于复杂配置，考虑分步骤更新
3. **回滚机制**: 保留配置历史，支持快速回滚
4. **监控告警**: 监控配置重载的成功率和性能

## 技术优势

- **零停机时间**: 无需重启应用即可更新配置
- **实时响应**: 毫秒级的配置变更检测
- **安全可靠**: 原子性操作和错误恢复机制
- **高性能**: 优化的文件监控和内存管理
- **易于扩展**: 灵活的回调机制和插件架构

配置热重载功能为 `zzz` 工具提供了企业级的配置管理能力，大大提升了开发和运维效率。