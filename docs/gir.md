# 有关 gir 的文档

## 有关 glib main loop

注意是 g.MainContextDefault 而不是 g.NewMainContext。

示例代码：

```go
mainCtx := g.MainContextDefault()
mainLoop := g.NewMainLoop(mainCtx, false)
go mainLoop.Run()
```

## 基本数据类型

| c type                  | c 32 | c 64 | go type                  |
| ----------------------- | ---- | ---- | ------------------------ |
| gchar / char            | 1    | 1    | int8                     |
| guchar / unsigned char  | 1    | 1    | byte / uint8             |
| gshort /short           | 2    | 2    | int16                    |
| gushort /unsigned short | 2    | 2    | uint16                   |
| gint / gboolean / int   | 4    | 4    | int32                    |
| guint / uint            | 4    | 4    | uint32                   |
| glong / long            | 4    | 8    | int                      |
| gulong / unsigned long  | 4    | 8    | uint                     |
| gpointer / void*        | 4    | 8    | unsafe.Pointer / uintptr |
| long long               | 8    | 8    | int64                    |
| unsigned long long      | 8    | 8    | uint64                   |
| float                   | 4    | 4    | float32                  |
| double                  | 8    | 8    | float64                  |

## 参考项目

- [gotk3](https://github.com/gotk3/gotk3)
- [mattn/go-gtk](https://github.com/mattn/go-gtk)

## 重要文件

/usr/share/gir-1.0/gir-1.2.rnc