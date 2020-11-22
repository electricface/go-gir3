/*
 * Copyright (C) 2019 ~ 2020 Uniontech Software Technology Co.,Ltd
 *
 * Author:
 *
 * Maintainer:
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package gtk

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk-a11y.h>
#include <gtk/gtk.h>
#include <gtk/gtkx.h>

GtkWidget *
_gtk_message_dialog_new (GtkWindow *parent,
                        GtkDialogFlags flags,
                        GtkMessageType type,
                        GtkButtonsType buttons,
                        const gchar *message) {
	return gtk_message_dialog_new(parent, flags, type, buttons, message);
}
 */
import "C"
import (
	"fmt"
	"unsafe"
)

func NewMessageDialog(parent Window, flags DialogFlags, msgType MessageTypeEnum,
	buttons ButtonsTypeEnum, messageFormat string, args ...interface{}) (result MessageDialog) {
	msg := fmt.Sprintf(messageFormat, args...)
	cMsg := C.CString(msg)
	ret := C._gtk_message_dialog_new((*C.GtkWindow)(parent.P), C.GtkDialogFlags(flags),
		C.GtkMessageType(msgType), C.GtkButtonsType(buttons), cMsg)
	C.free(unsafe.Pointer(cMsg))
	result.P = unsafe.Pointer(ret)
	return
}