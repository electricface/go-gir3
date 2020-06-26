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

package xmlp

import (
	"strings"
)

const (
	xmlnsCore = "core"
	xmlnsC    = "c"
	xmlnsGlib = "glib"
)

type Space uint

const (
	SpaceInvalid Space = iota
	SpaceCore
	SpaceC
	SpaceGlib
)

func getSpace(s string) Space {
	arr := strings.Split(s, "/")
	length := len(arr)
	if length < 2 {
		return SpaceInvalid
	}
	dir := arr[length-2]
	switch dir {
	case xmlnsCore:
		return SpaceCore
	case xmlnsC:
		return SpaceC
	case xmlnsGlib:
		return SpaceGlib
	default:
		return SpaceInvalid
	}
}
