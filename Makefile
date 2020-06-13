girgen:
	go build -o girgen -v github.com/electricface/go-gir3/cmd/girgen

gen_all: glib-2.0 gobject-2.0 gio-2.0 gudev-1.0 atk-1.0 cairo-1.0 gdk-3.0 pango-1.0 gdk-pixbuf-2.0 gtk-3.0

glib-2.0:
	./girgen -n GLib -v 2.0

gobject-2.0:
	./girgen -n GObject -v 2.0

gio-2.0:
	./girgen -n Gio -v 2.0

gudev-1.0:
	./girgen -n GUdev -v 1.0

atk-1.0:
	./girgen -n Atk -v 1.0

cairo-1.0:
	./girgen -n cairo -v 1.0

gdk-3.0:
	./girgen -n Gdk -v 3.0

pango-1.0:
	./girgen -n Pango -v 1.0

gdk-pixbuf-2.0:
	./girgen -n GdkPixbuf -v 2.0

gtk-3.0:
	./girgen -n Gtk -v 3.0

gtksource-4:
	./girgen -n GtkSource -v 4

.PHONY: girgen

