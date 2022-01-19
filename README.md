# go-gir3

##基本操作

生成最重要的代码生成工具 girgen，执行命令：
```shell
make girgen
```

生成代码：不要设置 GIRGEN_SYNC_MODE 环境变量，然后执行命令，生成所有支持的库的代码：
```shell
make gen_all
```

## 更新目标仓库 gi 包

目标仓库: https://github.com/electricface/go-gir。
把本项目的 gi-lite 复制到目标仓库的 gi 去。
```shell
./girgen -sync-gi
# 或
make sync_gi
```

## 测试所有包能否编译
```shell
cd $GOPATH/src/github.com/electricface/go-gir
go build -v ./...
```

-----
旧文档：

执行命令 `make gen_g` 会调用 girgen 生成 glib 的代码到 `$GOPATH/src/github.com/electricface/go-gir/g-2.0` 目录，生成的文件包括 glib_auto.go, gio_auto.go, gobject_auto.go 。 如果  GIRGEN_SYNC_MODE 环境变量为空或者 dev，则从 $GOPATH/src/github.com/electricface/go-gir 复制手写代码（非 *_auto.go 的 go 文件）和配置文件（*config.json）到本项目的 lib.in 文件夹中；
如果这个环境变量值为 build，则从本项目的 lib.in 文件夹复制文件到 $GOPATH/src/github.com/linuxdeepin/go-gir 。

执行 `./girgen -sync-gi` 或 `make sync_gi` 会把本项目 gi-lite 文件夹的代码复制到 $GOPATH/src/github.com/electricface/go-gir/gi 文件夹。

总结有关代码复制问题，通常应该在本项目编写 gi-lite 文件夹里的代码，在 go-gir 项目编写生成库（比如 g-2.0）中的手写代码，这遵循了以前的写作习惯，并把所有手写代码存一份在本项目中，然后 go-gir 项目还能独立打包，不依赖于本项目的代码。

