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

package main

import (
	"flag"
	"log"

	"github.com/electricface/go-gir3/gi"
)

func checkFi(fi *gi.FunctionInfo) {
	name := fi.Name()
	log.Println("function:", name)

	num := fi.NumArg()
	for i := 0; i < num; i++ {
		argInfo := fi.Arg(i)
		transfer := argInfo.OwnershipTransfer()
		dir := argInfo.Direction()

		argName := argInfo.Name()
		argTypeInfo := argInfo.Type()

		tag := argTypeInfo.Tag()

		isPtr := argTypeInfo.IsPointer()

		log.Printf("arg %s, ts: %v, dir: %v, isPtr: %v, type.tag: %s", argName, transfer, dir, isPtr, tag)

		argTypeInfo.Unref()
		argInfo.Unref()
	}

	retTypeInfo := fi.ReturnType()
	retTransfer := fi.CallerOwns()
	log.Printf("return ts: %v isPtr: %v, tag: %v", retTransfer, retTypeInfo.IsPointer(), retTypeInfo.Tag())
	retTypeInfo.Unref()
}

var optNamespace string
var optVersion string

var showMap = make(map[string]bool)

func init() {
	flag.StringVar(&optNamespace, "n", "", "namespace")
	flag.StringVar(&optVersion, "v", "", "version")
}

func main() {
	showMap["struct"] = true
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
			if !showMap["func"] {
				break
			}

			log.Println(name, "FUNCTION")
			fi := gi.ToFunctionInfo(bi)
			checkFi(fi)
		case gi.INFO_TYPE_CALLBACK:
		case gi.INFO_TYPE_STRUCT:
			if !showMap["struct"] {
				break
			}
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
			if !showMap["object"] {
				break
			}
			log.Println(name, "OBJECT")
			oi := gi.ToObjectInfo(bi)
			num := oi.NumMethod()
			for j := 0; j < num; j++ {
				fi := oi.Method(j)
				checkFi(fi)
			}
		case gi.INFO_TYPE_INTERFACE:
			if !showMap["interface"] {
				break
			}
			log.Println(name, "INTERFACE")
			info := gi.ToInterfaceInfo(bi)
			num := info.NumMethod()
			for j := 0; j < num; j++ {
				fi := info.Method(j)
				checkFi(fi)
			}

		case gi.INFO_TYPE_CONSTANT:
		case gi.INFO_TYPE_UNION:
			if !showMap["union"] {
				break
			}
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
