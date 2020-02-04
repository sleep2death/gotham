// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gotham

import (
	"errors"
	"path"
	"reflect"
	"runtime"
	"strings"
)

func assert1(guard bool, text string) {
	if !guard {
		panic(text)
	}
}

func lastChar(str string) uint8 {
	if str == "" {
		panic("The length of the string can't be 0")
	}
	return str[len(str)-1]
}

func nameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := path.Join(absolutePath, relativePath)
	appendSlash := lastChar(relativePath) == '/' && lastChar(finalPath) != '/'
	if appendSlash {
		return finalPath + "/"
	}
	return finalPath
}

var ErrHybridPath error = errors.New("hybrid type path is not allowed.")

func fixPath(p string) (res string, err error) {
	hasSlash := strings.Contains(p, "/")
	hasDot := strings.Contains(p, ".")
	res = ""

	if hasSlash && hasDot {
		return res, ErrHybridPath
	} else if hasDot {
		res = strings.Replace(p, ".", "/", -1)
	} else {
		res = p
	}

	// res = path.Clean(res)

	if res[:1] != "/" {
		res = "/" + res
	}

	if res[len(res)-1:] == "/" && len(res) > 1 {
		res = res[:len(res)-1]
	}
	return res, nil
}
