package main

import (
	"testing"

	"github.com/spiral/cascade"
	"github.com/spiral/cascade/tests/foo1"
	"github.com/spiral/cascade/tests/foo2"
	"github.com/spiral/cascade/tests/foo3"
	"github.com/spiral/cascade/tests/foo4"
)

func TestCascade_Init(t *testing.T) {
	c, err := cascade.NewContainer(cascade.TraceLevel)
	if err != nil {
		t.Fatal(err)
	}

	// foo4.S4 provides foo4.DB dependency, similar to the foo2.DB
	err = c.Register(&foo4.S4{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Register(&foo2.S2{})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Register(&foo3.S3{})
	if err != nil {
		t.Fatal(err)
	}
	err = c.Register(&foo1.S1{})
	if err != nil {
		t.Fatal(err)
	}

	err = c.Init()
	if err != nil {
		t.Fatal(err)
	}
}
