package main

import (
	"flag"
	"github.com/electricface/go-gir3/gi"
	"log"
)

func checkFi(fi *gi.FunctionInfo) {
	name := fi.Name()
	log.Println("function:", name)

	num := fi.NumArg()
	for i := 0;i < num; i++ {
		argInfo := fi.Arg(i)
		dir := argInfo.Direction()

		argName := argInfo.Name()
		argTypeInfo := argInfo.Type()

		tag := argTypeInfo.Tag()

		isPtr := argTypeInfo.IsPointer()

		log.Printf("arg %s, dir: %v, isPtr: %v, type.tag: %s", argName, dir, isPtr, tag)

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	retTypeInfo := fi.ReturnType()
	log.Printf("return isPtr: %v, tag: %v", retTypeInfo.IsPointer(), retTypeInfo.Tag())
	retTypeInfo.Unref()
}

var optNamespace string
var optVersion string

func init() {
	flag.StringVar(&optNamespace, "n", "", "namespace")
	flag.StringVar(&optVersion, "v", "", "version")
}

func main() {
	flag.Parse()

	repo := gi.DefaultRepository()
	_, err := repo.Require(optNamespace, optVersion, gi.REPOSITORY_LOAD_FLAG_LAZY)
	if err != nil {
		log.Fatal(err)
	}

	num := repo.NumInfo(optNamespace)
	for i := 0; i < num; i++ {
		bi := repo.Info(optNamespace, i)
		name := bi.Name()
		switch bi.Type() {
		case gi.INFO_TYPE_FUNCTION:
			log.Println(name, "FUNCTION")
			fi := gi.ToFunctionInfo(bi)
			checkFi(fi)

		case gi.INFO_TYPE_CALLBACK:
		case gi.INFO_TYPE_STRUCT:
			log.Println(name, "STRUCT")

			si := gi.ToStructInfo(bi)
			num := si.NumMethod()
			for j := 0; j < num; j++ {
				fi := si.Method(j)
				checkFi(fi)
			}

		case gi.INFO_TYPE_BOXED:
		case gi.INFO_TYPE_ENUM:
		case gi.INFO_TYPE_FLAGS:
		case gi.INFO_TYPE_OBJECT:
			log.Println(name, "OBJECT")
			oi := gi.ToObjectInfo(bi)
			num := oi.NumMethod()
			for j := 0; j < num; j++ {
				fi := oi.Method(j)
				checkFi(fi)
			}
		case gi.INFO_TYPE_INTERFACE:
			log.Println(name, "INTERFACE")
			info := gi.ToInterfaceInfo(bi)
			num := info.NumMethod()
			for j := 0; j < num; j++ {
				fi := info.Method(j)
				checkFi(fi)
			}

		case gi.INFO_TYPE_CONSTANT:
		case gi.INFO_TYPE_UNION:
			log.Println(name, "UNION")

		case gi.INFO_TYPE_VALUE:
		case gi.INFO_TYPE_SIGNAL:
		case gi.INFO_TYPE_VFUNC:
		case gi.INFO_TYPE_PROPERTY:
		case gi.INFO_TYPE_FIELD:
		case gi.INFO_TYPE_ARG:
		case gi.INFO_TYPE_TYPE:
		}
		bi.Unref()
	}



	//if name == "File" {
	//	log.Println(bi.Type().String())
	//	ifcInfo := gi.ToInterfaceInfo(bi)
	//	numMethod := ifcInfo.NumMethod()
	//	for j := 0; j < numMethod; j++ {
	//		funcInfo := ifcInfo.Method(j)
	//		if funcInfo.Name() == "new_for_path" {
	//			// test it
	//			var ret gi.Argument
	//			path1 := "/home/tp1"
	//			_ = path1
	//			//err := funcInfo.Invoke([]gi.Argument{ gi.NewStringArgument(strPtr(&path1)) }, []gi.Argument{}, &ret)
	//			spew.Dump(ret)
	//			//if err != nil {
	//			//	log.Fatal(err)
	//			//}
	//			log.Println("invoke success")
	//		}
	//	}
	//	return
	//}
}


//switch bi.Type() {
//case gi.INFO_TYPE_FUNCTION:
//case gi.INFO_TYPE_CALLBACK:
//case gi.INFO_TYPE_STRUCT:
//case gi.INFO_TYPE_BOXED:
//case gi.INFO_TYPE_ENUM:
//case gi.INFO_TYPE_FLAGS:
//case gi.INFO_TYPE_OBJECT:
//case gi.INFO_TYPE_INTERFACE:
//case gi.INFO_TYPE_CONSTANT:
//case gi.INFO_TYPE_UNION:
//case gi.INFO_TYPE_VALUE:
//case gi.INFO_TYPE_SIGNAL:
//case gi.INFO_TYPE_VFUNC:
//case gi.INFO_TYPE_PROPERTY:
//case gi.INFO_TYPE_FIELD:
//case gi.INFO_TYPE_ARG:
//case gi.INFO_TYPE_TYPE:
//}
