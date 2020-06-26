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
 */

package gi

/*
#include <girepository.h>
*/
import "C"
import "unsafe"

// C.GIArgument is [8]byte
type Argument [8]byte

func NewUint8Argument(v uint8) (arg Argument) {
	*(*uint8)(unsafe.Pointer(&arg)) = v
	return
}

func NewInt8Argument(v int8) (arg Argument) {
	*(*int8)(unsafe.Pointer(&arg)) = v
	return
}

func NewBoolArgument(v bool) (arg Argument) {
	var val C.gboolean
	if v {
		val = 1
	}
	*(*C.gboolean)(unsafe.Pointer(&arg)) = val
	return
}

func NewUint16Argument(v uint16) (arg Argument) {
	*(*uint16)(unsafe.Pointer(&arg)) = v
	return
}

func NewInt16Argument(v int16) (arg Argument) {
	*(*int16)(unsafe.Pointer(&arg)) = v
	return
}

func NewUint32Argument(v uint32) (arg Argument) {
	*(*uint32)(unsafe.Pointer(&arg)) = v
	return
}

func NewInt32Argument(v int32) (arg Argument) {
	*(*int32)(unsafe.Pointer(&arg)) = v
	return
}

func NewUint64Argument(v uint64) (arg Argument) {
	*(*uint64)(unsafe.Pointer(&arg)) = v
	return
}

func NewInt64Argument(v int64) (arg Argument) {
	*(*int64)(unsafe.Pointer(&arg)) = v
	return
}

func NewFloatArgument(v float32) (arg Argument) {
	*(*C.gfloat)(unsafe.Pointer(&arg)) = (C.gfloat)(v)
	return
}

func NewDoubleArgument(v float64) (arg Argument) {
	*(*C.gdouble)(unsafe.Pointer(&arg)) = (C.gdouble)(v)
	return
}

func NewShortArgument(v int16) (arg Argument) {
	*(*C.gshort)(unsafe.Pointer(&arg)) = (C.gshort)(v)
	return
}

func NewUShortArgument(v int16) (arg Argument) {
	*(*C.gushort)(unsafe.Pointer(&arg)) = (C.gushort)(v)
	return
}

func NewIntArgument(v int) (arg Argument) {
	*(*C.gint)(unsafe.Pointer(&arg)) = (C.gint)(v)
	return
}

func NewUintArgument(v uint) (arg Argument) {
	*(*C.guint)(unsafe.Pointer(&arg)) = (C.guint)(v)
	return
}

func NewLongArgument(v int64) (arg Argument) {
	*(*C.glong)(unsafe.Pointer(&arg)) = (C.glong)(v)
	return
}

func NewULongArgument(v uint64) (arg Argument) {
	*(*C.gulong)(unsafe.Pointer(&arg)) = (C.gulong)(v)
	return
}

func NewSSizeArgument(v int64) (arg Argument) {
	*(*C.gssize)(unsafe.Pointer(&arg)) = (C.gssize)(v)
	return
}

func NewSizeArgument(v uint64) (arg Argument) {
	*(*C.gsize)(unsafe.Pointer(&arg)) = (C.gsize)(v)
	return
}

func NewStringArgument(v unsafe.Pointer) (arg Argument) {
	*(**C.gchar)(unsafe.Pointer(&arg)) = (*C.gchar)(v)
	return
}

func NewPointerArgument(v unsafe.Pointer) (arg Argument) {
	*(*C.gpointer)(unsafe.Pointer(&arg)) = (C.gpointer)(v)
	return
}

func (arg Argument) Bool() bool {
	val := *(*C.gboolean)(unsafe.Pointer(&arg))
	return val != 0
}

func (arg Argument) Int8() int8 {
	return *(*int8)(unsafe.Pointer(&arg))
}

func (arg Argument) Uint8() uint8 {
	return *(*uint8)(unsafe.Pointer(&arg))
}

func (arg Argument) Int16() int16 {
	return *(*int16)(unsafe.Pointer(&arg))
}

func (arg Argument) Uint16() uint16 {
	return *(*uint16)(unsafe.Pointer(&arg))
}

func (arg Argument) Int32() int32 {
	return *(*int32)(unsafe.Pointer(&arg))
}

func (arg Argument) Uint32() uint32 {
	return *(*uint32)(unsafe.Pointer(&arg))
}

func (arg Argument) Int64() int64 {
	return *(*int64)(unsafe.Pointer(&arg))
}

func (arg Argument) Uint64() uint64 {
	return *(*uint64)(unsafe.Pointer(&arg))
}

func (arg Argument) Float() float32 {
	return float32(*(*C.gfloat)(unsafe.Pointer(&arg)))
}

func (arg Argument) Double() float64 {
	return float64(*(*C.gdouble)(unsafe.Pointer(&arg)))
}

func (arg Argument) Short() int16 {
	return int16(*(*C.gshort)(unsafe.Pointer(&arg)))
}

func (arg Argument) UShort() uint16 {
	return uint16(*(*C.gushort)(unsafe.Pointer(&arg)))
}

func (arg Argument) Int() int {
	return int(*(*C.gint)(unsafe.Pointer(&arg)))
}

func (arg Argument) Uint() uint {
	return uint(*(*C.guint)(unsafe.Pointer(&arg)))
}

func (arg Argument) Long() int64 {
	return int64(*(*C.glong)(unsafe.Pointer(&arg)))
}

func (arg Argument) ULong() uint64 {
	return uint64(*(*C.gulong)(unsafe.Pointer(&arg)))
}

func (arg Argument) SSize() int64 {
	return int64(*(*C.gssize)(unsafe.Pointer(&arg)))
}

func (arg Argument) Size() uint64 {
	return uint64(*(*C.gsize)(unsafe.Pointer(&arg)))
}

func (arg Argument) String() StrPtr {
	return StrPtr{*(*unsafe.Pointer)(unsafe.Pointer(&arg))}
}

func (arg Argument) Pointer() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&arg))
}
