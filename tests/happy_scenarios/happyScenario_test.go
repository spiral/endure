package happy_scenarios

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	endure "github.com/spiral/endure/pkg/container"
	"github.com/spiral/endure/tests/happy_scenarios/plugin1"
	"github.com/spiral/endure/tests/happy_scenarios/plugin2"
	"github.com/spiral/endure/tests/happy_scenarios/plugin3"
	"github.com/spiral/endure/tests/happy_scenarios/plugin4"
	"github.com/spiral/endure/tests/happy_scenarios/plugin5"
	"github.com/spiral/endure/tests/happy_scenarios/plugin6"
	"github.com/spiral/endure/tests/happy_scenarios/plugin7"
	primitive "github.com/spiral/endure/tests/happy_scenarios/plugin8"
	plugin12 "github.com/spiral/endure/tests/happy_scenarios/provided_value_but_need_pointer/plugin1"
	plugin22 "github.com/spiral/endure/tests/happy_scenarios/provided_value_but_need_pointer/plugin2"
	"github.com/stretchr/testify/assert"
)

func TestEndure_DifferentLogLevels(t *testing.T) {
	testLog(t, endure.DebugLevel)
	testLog(t, endure.WarnLevel)
	testLog(t, endure.InfoLevel)
	testLog(t, endure.FatalLevel)
	testLog(t, endure.ErrorLevel)
	testLog(t, endure.DPanicLevel)
	testLog(t, endure.PanicLevel)
}

func testLog(t *testing.T, level endure.Level) {
	c, err := endure.NewContainer(nil, endure.SetLogLevel(level))
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))
	assert.NoError(t, c.Init())

	res, err := c.Serve()
	assert.NoError(t, err)

	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)
	assert.NoError(t, c.Stop())
}

func TestEndure_Init_OK(t *testing.T) {
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))

	assert.NoError(t, c.Init())

	res, err := c.Serve()
	assert.NoError(t, err)
	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)
	assert.NoError(t, c.Stop())
}

func TestEndure_DoubleInitDoubleServe_OK(t *testing.T) {
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))

	assert.NoError(t, c.Init())
	assert.Error(t, c.Init())

	_, err = c.Serve()
	assert.NoError(t, err)
	res, err := c.Serve()
	assert.Error(t, err)
	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)
	assert.NoError(t, c.Stop())
}

func TestEndure_WrongOrder(t *testing.T) {
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.Error(t, c.Stop()) //recognizer: can't transition from state: Uninitialized by event Stop
	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))

	_, err = c.Serve()
	assert.Error(t, err)

	assert.NoError(t, c.Init())
	assert.Error(t, c.Init())

	res, err := c.Serve()
	assert.NoError(t, err)
	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)
	assert.NoError(t, c.Stop())
}

func TestEndure_Init_1_Element(t *testing.T) {
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin7.Plugin7{}))
	assert.NoError(t, c.Init())

	res, err := c.Serve()
	assert.NoError(t, err)

	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)

	assert.NoError(t, c.Stop())
}

func TestEndure_ProvidedValueButNeedPointer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			println("test should panic")
		}
	}()
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin12.Plugin1{}))
	assert.NoError(t, c.Register(&plugin22.Plugin2{}))
	assert.NoError(t, c.Init())

	res, err := c.Serve()
	assert.NoError(t, err)

	go func() {
		for r := range res {
			if r.Error != nil {
				assert.NoError(t, r.Error)
				return
			}
		}
	}()

	time.Sleep(time.Second * 2)

	assert.NoError(t, c.Stop())
}

func TestEndure_PrimitiveTypes(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			println("test should panic")
		}
	}()
	c, err := endure.NewContainer(nil)
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&primitive.Plugin8{}))
	assert.NoError(t, c.Init())

	_, _ = c.Serve()
	assert.NoError(t, err)

	assert.NoError(t, c.Stop())
}

func TestEndure_VisualizeFile(t *testing.T) {
	golden :=
		`digraph endure {
	rankdir=TB;
	graph [compound=true];
		"plugin4.S4" -> "plugin5.S5";
		"plugin4.S4" -> "plugin6.S6Interface";
		"plugin2.S2" -> "plugin4.S4";
		"plugin3.S3" -> "plugin4.S4";
		"plugin3.S3" -> "plugin2.S2";
		"plugin1.S1" -> "plugin4.S4";
		"plugin1.S1" -> "plugin2.S2";
}
`
	filename := "graph.txt"
	c, err := endure.NewContainer(nil, endure.Visualize(endure.File, filename), endure.SetLogLevel(endure.ErrorLevel))
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))

	assert.NoError(t, c.Init())

	// should exist
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.Remove(filename)
		if err != nil {
			assert.Fail(t, "can't delete file")
		}
	}()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	// string(data) is safe because there are non 1-byte symbols in the file
	assert.Equal(t, string(data), golden)
}

func TestEndure_VisualizeStdOut(t *testing.T) {
	c, err := endure.NewContainer(nil, endure.Visualize(endure.StdOut, ""), endure.SetLogLevel(endure.PanicLevel))
	assert.NoError(t, err)

	assert.NoError(t, c.Register(&plugin4.S4{}))
	assert.NoError(t, c.Register(&plugin2.S2{}))
	assert.NoError(t, c.Register(&plugin3.S3{}))
	assert.NoError(t, c.Register(&plugin1.S1{}))
	assert.NoError(t, c.Register(&plugin5.S5{}))
	assert.NoError(t, c.Register(&plugin6.S6Interface{}))

	assert.NoError(t, c.Init())
}
