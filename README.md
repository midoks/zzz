# zzz

Go实时开发小工具

```
原来用的bee，很好用哈。最近开发项目遇到一个需求。需求是编译前预处理一下。

```

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
action:
  before:
  - find . -name ".DS_Store" -print -delete
  after:
  - echo "zzz end"

```

- dirfilter:不监控目录
- ext:监控文件后缀
- action.before:执行前处理
- action.after:执行后处理
