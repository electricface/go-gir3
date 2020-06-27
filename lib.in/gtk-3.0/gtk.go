package gtk

import gi "github.com/electricface/go-gir3/gi-lite"
import "unsafe"
import "log"

func Init(argc int, argv int) {
	iv, err := _I.Get(3703, "init", "", -1, -1, gi.INFO_TYPE_FUNCTION, 0)
	if err != nil {
		log.Println("WARN:", err)
		return
	}
	var outArgs [2]gi.Argument
	arg_argc := gi.NewPointerArgument(unsafe.Pointer(&outArgs[0]))
	arg_argv := gi.NewPointerArgument(unsafe.Pointer(&outArgs[1]))
	args := []gi.Argument{arg_argc, arg_argv}
	iv.Call(args, nil, &outArgs[0])
}
