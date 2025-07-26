# zzz

🚀 高性能 Go/Rust 实时开发工具

一个轻量级、高效的实时编译工具，支持 Go 和 Rust 项目的自动构建和热重载。

## ✨ 特性

- 🔥 **实时热重载** - 文件变化时自动重新编译和运行
- 🎯 **智能防抖** - 避免频繁的重复构建
- 📊 **性能监控** - 实时显示构建时间和内存使用情况
- 🛡️ **进程管理** - 优雅的进程启动和终止
- ⚙️ **灵活配置** - 支持自定义构建参数和钩子
- 🌍 **跨平台** - 支持 Windows、macOS 和 Linux
- 🦀 **多语言** - 同时支持 Go 和 Rust 项目

### 安装

```bash
go install github.com/midoks/zzz@latest
```

### 直接运行

```bash
zzz run
```

### 创建配置文件

```bash
zzz new
```

- .zzz.yaml

```
title: zzz
frequency: 3
lang: go
dirfilter:
- tmp
- .git
- public
- scripts
- vendor
- logs
- templates
ext:
- go
enablerun: true
action:
  before:
  - find . -name ".DS_Store" -print -delete
  after:
  - echo "zzz end"
  exit:
  - exho "exit"
link: https://github.com/midoks/zzz

```

- frequency:编译时间间隔,单位秒
- lang: 仅支持go|rust
- dirfilter:不监控目录
- ext:监控文件后缀
- action.before:执行前处理
- action.after:执行后处理
- action.exit:退出执行
- enablerun:是否直接执行[go]
