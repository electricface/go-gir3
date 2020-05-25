girgen:
	go build -o girgen -v github.com/electricface/go-gir3/cmd/girgen

glib-2.0:
	./girgen -n GLib -v 2.0  -p glib -d $(GOPATH)/src/github.com/electricface/go-gir/glib-2.0

.PHONY: girgen

