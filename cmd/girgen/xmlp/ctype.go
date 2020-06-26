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
	"errors"
	"strings"
)

type CType struct {
	Name       string
	IsUnsigned bool
	NumStar    int
}

// 比如:
// char
// char*
// const char* const*
func ParseCType(ctype string) (*CType, error) {
	numStar := strings.Count(ctype, "*")
	ctype = strings.Replace(ctype, "*", " ", -1) // remove all *
	fields := strings.Fields(ctype)

	var fieldsTemp []string
	for _, val := range fields {
		if val != "const" {
			fieldsTemp = append(fieldsTemp, val)
		} // else is const, removed
	}
	fields = fieldsTemp
	if len(fields) == 0 {
		return nil, errors.New("fields empty")
	}

	lastField := fields[len(fields)-1]

	var isUnsigned bool
	for _, f := range fields[:len(fields)-1] {
		if f == "unsigned" {
			isUnsigned = true
		}
	}

	return &CType{
		Name:       lastField,
		NumStar:    numStar,
		IsUnsigned: isUnsigned,
	}, nil
}

func (ct *CType) CgoNotation() string {
	ret := strings.Repeat("*", ct.NumStar)
	name := ct.Name
	if ct.IsUnsigned {
		if ct.Name == "int" {
			name = "uint"
		} else {
			panic("unsupported unsigned type " + ct.Name)
		}
	}

	ret = ret + "C." + name
	return ret
}

func (ct *CType) Elem() *CType {
	numStar := ct.NumStar - 1
	if numStar < 0 {
		panic("assert failed numStr >= 0")
	}

	return &CType{
		Name:       ct.Name,
		NumStar:    numStar,
		IsUnsigned: ct.IsUnsigned,
	}
}
