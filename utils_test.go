package gotham

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLastChar(t *testing.T) {
	assert.Equal(t, uint8('a'), lastChar("hola"))
	assert.Equal(t, uint8('s'), lastChar("adios"))
	assert.Panics(t, func() { lastChar("") })
}

func TestFunctionName(t *testing.T) {
	assert.Regexp(t, `^(.*/vendor/)?github.com/sleep2death/gotham.somefunction$`, nameOfFunction(somefunction))
}

func somefunction() {
	// this empty function is used by TestFunctionName()
}

func TestJoinPaths(t *testing.T) {
	assert.Equal(t, "", joinPaths("", ""))
	assert.Equal(t, "/", joinPaths("", "/"))
	assert.Equal(t, "/a", joinPaths("/a", ""))
	assert.Equal(t, "/a/", joinPaths("/a/", ""))
	assert.Equal(t, "/a/", joinPaths("/a/", "/"))
	assert.Equal(t, "/a/", joinPaths("/a", "/"))
	assert.Equal(t, "/a/hola", joinPaths("/a", "/hola"))
	assert.Equal(t, "/a/hola", joinPaths("/a/", "/hola"))
	assert.Equal(t, "/a/hola/", joinPaths("/a/", "/hola/"))
	assert.Equal(t, "/a/hola/", joinPaths("/a/", "/hola//"))
}

func TestClearPath(t *testing.T) {
	path := "/abc/def.a"
	_, err := fixPath(path)
	assert.Equal(t, ErrHybridPath, err)

	path = "abc/hello"
	res, err := fixPath(path)
	assert.Equal(t, "/abc/hello", res)

	path = "abc/hello/"
	res, err = fixPath(path)
	assert.Equal(t, "/abc/hello", res)

	path = "abc-hello"
	res, err = fixPath(path)
	assert.Equal(t, "/abc-hello", res)

	path = "gotham.hello.world"
	res, err = fixPath(path)
	assert.Equal(t, "/gotham/hello/world", res)

	path = "gotham.hello.world."
	res, err = fixPath(path)
	assert.Equal(t, "/gotham/hello/world", res)
}
