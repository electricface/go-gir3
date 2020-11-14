
重要文件
/usr/share/gir-1.0/gir-1.2.rnc

## 有关 glib main loop

注意是 g.MainContextDefault 而不是 g.NewMainContext。

示例代码：
```go
mainCtx := g.MainContextDefault()
mainLoop := g.NewMainLoop(mainCtx, false)
go mainLoop.Run()
```