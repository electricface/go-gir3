export GIR_PKG_PATH := github.com/linuxdeepin/go-gir
GOPATH_DIR=gopath
G_DIR=%GOPATH%/src/$(GIR_PKG_PATH)/g-2.0
GOPKG_PREFIX = github.com/electricface/go-gir3
GOBUILD = go build $(GO_BUILD_FLAGS)

.PHONY: all prepare
# %GOPATH% 会在 girgen 中替换成 GOPATH 的第一个
git_project_root=$(shell git rev-parse --show-toplevel)

all: prepare girgen

prepare:
	@mkdir -p out/bin
	@if [ ! -d ${GOPATH_DIR}/src/${GOPKG_PREFIX} ]; then \
         mkdir -p ${GOPATH_DIR}/src/$(dir ${GOPKG_PREFIX}); \
         ln -sf ../../../.. ${GOPATH_DIR}/src/${GOPKG_PREFIX}; \
         fi

girgen:
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o $@ -v github.com/electricface/go-gir3/cmd/girgen

gen_array_code:
	go build -o gen_array_code -v github.com/electricface/go-gir3/cmd/gen_array_code
	./gen_array_code > $(git_project_root)/gi-lite/arr_auto.go
	go fmt $(git_project_root)/gi-lite/arr_auto.go
	go build -v github.com/electricface/go-gir3/gi-lite

sync_gi:
	./girgen -sync-gi

gen_g: glib-2.0 gobject-2.0 gio-2.0

gen_gtk: atk-1.0 cairo-1.0 gdk-3.0 pango-1.0 gdk-pixbuf-2.0 gdk-pixdata-2.0 gtk-3.0 gtksource-4

gen_other: gudev-1.0 pangocairo-1.0 vte-2.91 girepository-2.0 rsvg-2.0 poppler-0.18 atspi-2.0 udisks-2.0 gst-1.0 gstbase-1.0 gstcontroller-1.0 gstnet-1.0

gen_all: sync_gi gen_g gen_gtk gen_other


glib-2.0:
	./girgen -n GLib -v 2.0 -p g -f $(G_DIR)/glib_auto.go -c glib-config.json
	# libgirepository1.0-dev gir1.2-glib-2.0
	# dev 包放 .gir 文件，gir1.2 包放 typelib 文件
	# .gir 文件一般放在 /usr/share/gir-1.0/
	# .typelib 文件一般放在 /usr/lib/x86_64-linux-gnu/girepository-1.0 文件夹

gobject-2.0:
	./girgen -n GObject -v 2.0 -p g -f $(G_DIR)/gobject_auto.go -c gobject-config.json
	# libgirepository1.0-dev gir1.2-glib-2.0

gio-2.0:
	./girgen -n Gio -v 2.0 -p g -f $(G_DIR)/gio_auto.go
	# libgirepository1.0-dev gir1.2-glib-2.0

gudev-1.0:
	./girgen -n GUdev -v 1.0
	# libgudev-1.0-dev gir1.2-gudev-1.0

atk-1.0:
	./girgen -n Atk -v 1.0
	# libatk1.0-dev gir1.2-atk-1.0

cairo-1.0:
	./girgen -n cairo -v 1.0
	# libgirepository1.0-dev

gdk-3.0:
	./girgen -n Gdk -v 3.0
	#  libgtk-3-dev gir1.2-gtk-3.0

pango-1.0:
	./girgen -n Pango -v 1.0
	# libpango1.0-dev gir1.2-pango-1.0

pangocairo-1.0:
	./girgen -n PangoCairo -v 1.0
	# libpango1.0-dev gir1.2-pango-1.0

gdk-pixbuf-2.0:
	./girgen -n GdkPixbuf -v 2.0
	# gir1.2-gtk-3.0 gir1.2-gdkpixbuf-2.0

gdk-pixdata-2.0:
	./girgen -n GdkPixdata -v 2.0
	# gir1.2-gtk-3.0 gir1.2-gdkpixbuf-2.0

gtk-3.0:
	./girgen -n Gtk -v 3.0
	# libgtk-3-dev gir1.2-gtk-3.0

gtksource-4:
	./girgen -n GtkSource -v 4
	# libgtksourceview-4-dev gir1.2-gtksource-4

vte-2.91:
	./girgen -n Vte -v 2.91
	# libvte-2.91-dev gir1.2-vte-2.91

#gtop-2.0:
	#./girgen -n GTop -v 2.0
	# libgtop2-dev gir1.2-gtop-2.0
	# 调用 girgen 时有错误 XML syntax error on line 38: illegal character code U+0004

girepository-2.0:
	./girgen -n GIRepository -v 2.0
	# libgirepository1.0-dev

rsvg-2.0:
	./girgen -n Rsvg -v 2.0
	# librsvg2-dev gir1.2-rsvg-2.0

poppler-0.18:
	./girgen -n Poppler -v 0.18
	# libpoppler-glib-dev gir1.2-poppler-0.18

atspi-2.0:
	./girgen -n Atspi -v 2.0
	# libatspi2.0-dev gir1.2-atspi-2.0

#wnck-3.0:
#	./girgen -n Wnck -v 3.0
#	# libwnck-3-dev gir1.2-wnck-3.0
# 编译的 github.com/electricface/go-gir/wnck-3.0 的时候有报错提示
#In file included from /usr/include/libwnck-3.0/libwnck/libwnck.h:26,
#                 from wnck-3.0/wnck_auto.go:5:
#/usr/include/libwnck-3.0/libwnck/window.h:30:2: error: #error "libwnck should only be used if you understand that it's subject to frequent change, and is not supported as a fixed API/ABI or as part of the platform"
# #error "libwnck should only be used if you understand that it's subject to frequent change, and is not supported as a fixed API/ABI or as part of the platform"
#  ^~~~~

udisks-2.0:
	./girgen -n UDisks -v 2.0
	# libudisks2-dev gir1.2-udisks-2.0

gst-1.0:
	./girgen -n Gst -v 1.0
	# libgstreamer1.0-dev gir1.2-gstreamer-1.0

gstbase-1.0:
	./girgen -n GstBase -v 1.0
	# libgstreamer1.0-dev gir1.2-gstreamer-1.0

gstcontroller-1.0:
	./girgen -n GstController -v 1.0
	# libgstreamer1.0-dev gir1.2-gstreamer-1.0

gstnet-1.0:
	./girgen -n GstNet -v 1.0
	# libgstreamer1.0-dev gir1.2-gstreamer-1.0

.PHONY: girgen gen_array_code
