# zzz

Go实时开发小工具

```
原来用的bee，很好用哈。最近开发项目遇到一个需求。需求是编译前预处理一下，所以搞一下。

```

支持golang,rust实时编译

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
- dirfilter:不监控目录
- ext:监控文件后缀
- action.before:执行前处理
- action.after:执行后处理
- action.exit:退出执行
- enablerun:是否直接执行[go]
