girgen:
	go build -o girgen -v github.com/electricface/go-gir3/cmd/girgen

glib-2.0:
	./girgen -n GLib -v 2.0  -p glib -d $(GOPATH)/src/github.com/electricface/go-gir/glib-2.0


gobject-2.0:
	./girgen -n GObject -v 2.0  -p gobject -d $(GOPATH)/src/github.com/electricface/go-gir/gobject-2.0

gio-2.0:
	./girgen -n Gio -v 2.0  -p gio -d $(GOPATH)/src/github.com/electricface/go-gir/gio-2.0

.PHONY: girgen

