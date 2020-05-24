package main

import (
	. "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) {
	TestingT(t)
}

type SourceFileTestSuite struct {
}

var _ = Suite(&SourceFileTestSuite{})

func (s *SourceFileTestSuite) TestWriteStr(c *C) {
	//setGirProjectRoot("go-gir")
	sf := NewSourceFile("glib")
	sf.GoBody.P("/*go:.util*/ hello world /*go:unsafe*/")
	sf.Print()
}
