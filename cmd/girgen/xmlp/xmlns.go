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
