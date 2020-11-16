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

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	args := []interface{}{1, 2, 3}
	var a int
	var b uint
	var c uint16
	err := Store(args, &a, &b, &c)
	assert.EqualValues(t, 1, a)
	assert.EqualValues(t, 2, b)
	assert.EqualValues(t, 3, c)
	assert.Nil(t, err)
}

type Obj struct {
	a int
	P unsafe.Pointer
}

type Foo struct {
	b byte
	P unsafe.Pointer
}

func TestStoreInterfaces(t *testing.T) {
	a := Uint2Ptr(1)
	var b unsafe.Pointer
	err := storeInterfaces(a, &b)
	assert.Equal(t, a, b)
	assert.Nil(t, err)

	// 应对 unsafe.Pointer 转化为 gdk.Event
	var o1 Obj
	err = storeInterfaces(a, &o1)
	assert.Equal(t, a, o1.P)
	assert.Nil(t, err)

	// 应对 g.Object 转化成 gtk.Window
	foo := Foo{P: Uint2Ptr(2)}
	var o2 Obj
	err = storeInterfaces(foo, &o2)
	assert.Equal(t, foo.P, o2.P)
	assert.Nil(t, err)
}

func TestStoreStruct(t *testing.T) {
	args := []interface{}{1, 2, 3}
	var s struct {
		A int
		B uint
		C uint16
	}
	err := StoreStruct(args, &s)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, s.A)
	assert.EqualValues(t, 2, s.B)
	assert.EqualValues(t, 3, s.C)

	args = []interface{}{
		Obj{P: Uint2Ptr(1)},
		Uint2Ptr(2),
	}
	var s1 struct {
		A Foo
		B Obj
	}
	err = StoreStruct(args, &s1)
	assert.Nil(t, err)
	assert.Equal(t, Uint2Ptr(1), s1.A.P)
	assert.Equal(t, Uint2Ptr(2), s1.B.P)
}
