package cnfgfile_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"golift.io/cnfg"
	"golift.io/cnfg/cnfgfile"
)

type testStruct struct {
	PointerSlice  []*testSubConfig `json:"pslice" xml:"pslice" yaml:"pslice" toml:"pslice"`
	StructSlice   []testSubConfig  `json:"sslice" xml:"sslice" yaml:"sslice" toml:"sslice"`
	Struct        testSubConfig    `json:"struct" xml:"struct" yaml:"struct" toml:"struct"`
	PointerStruct *testSubConfig   `json:"pstruct" xml:"pstruct" yaml:"pstruct" toml:"pstruct"`
	// These dont get targeted during unmarhsal (not in the files).
	PointerSlice2  []*testSubConfig `json:"pslice2" xml:"pslice2" yaml:"pslice2" toml:"pslice2"`
	StructSlice2   []testSubConfig  `json:"sslice2" xml:"sslice2" yaml:"sslice2" toml:"sslice2"`
	Struct2        testSubConfig    `json:"struct2" xml:"struct2" yaml:"struct2" toml:"struct2"`
	PointerStruct2 *testSubConfig   `json:"pstruct2" xml:"pstruct2" yaml:"pstruct2" toml:"pstruct2"`
}

type testSubConfig struct {
	Bool    bool     `json:"bool" xml:"bool" yaml:"bool" toml:"bool"`
	Int     int64    `json:"int" xml:"int" yaml:"int" toml:"int"`
	StringP *string  `json:"string" xml:"string" yaml:"string" toml:"string"`
	FloatP  *float64 `json:"float" xml:"float" yaml:"float" toml:"float"`
}

func testUnmarshalValues(assert *assert.Assertions, config *testStruct, err error, from string) {
	from += " "

	assert.Nil(err, "there should not be an error reading the test file")
	// PointerSlice
	assert.Equal(1, len(config.PointerSlice), from+"pointerslice is too short")
	assert.EqualValues(true, config.PointerSlice[0].Bool, from+"the boolean was true")
	assert.EqualValues(123.4567, *config.PointerSlice[0].FloatP, from+"the float64 was set to 123.4567")
	assert.EqualValues(0, config.PointerSlice[0].Int, from+"int was not set so should be zero")
	assert.Nil(config.PointerSlice[0].StringP, from+"the string pointer was not set so should remain nil")

	// StructSlice
	assert.Equal(1, len(config.StructSlice), from+"pointerslice is too short")
	assert.EqualValues(false, config.StructSlice[0].Bool, from+"the boolean was missing and should be false")
	assert.Nil(config.StructSlice[0].FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(123, config.StructSlice[0].Int, from+"int was set to 123")
	assert.EqualValues("foo", *config.StructSlice[0].StringP, from+"the string was set to foo")

	// Struct
	assert.EqualValues(false, config.Struct.Bool, from+"the boolean was false and should be false")
	assert.Nil(config.Struct.FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(0, config.Struct.Int, from+"int was not set and must be 0")
	assert.Nil(config.Struct.StringP, from+"the string was missing and should be nil")

	// PointerStruct
	assert.NotNil(config.PointerStruct, from+"the pointer struct has values and must not be nil")
	assert.EqualValues(false, config.PointerStruct.Bool, from+"the boolean was missing and should be false")
	assert.Nil(config.PointerStruct.FloatP, from+"the float64 was missing and should be nil")
	assert.EqualValues(0, config.PointerStruct.Int, from+"int was not set and must be 0")
	assert.EqualValues("foo2", *config.PointerStruct.StringP, from+"the string was set to foo2")

	// PointerSlice2
	assert.Equal(0, len(config.PointerSlice2), from+"pointerslice2 is too long")
	// StructSlice2
	assert.Equal(0, len(config.StructSlice2), from+"structslice2 is too long")
	// Struct2
	assert.EqualValues(false, config.Struct2.Bool, from+"this must be zero value")
	assert.Nil(config.Struct2.FloatP, from+"this must be zero value")
	assert.EqualValues(0, config.Struct2.Int, from+"this must be zero value")
	assert.Nil(config.Struct2.StringP, from+"this must be zero value")
	// PointerStruct2
	assert.Nil(config.PointerStruct2, from+"pointer struct 2 must be nil")
}

func TestUnmarshalErrors(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	config := &testStruct{}

	err := cnfgfile.Unmarshal(config, "/etc/passwd")
	assert.NotNil(err, "there should be an error parsing a password file")

	err = cnfgfile.Unmarshal(config, "no file here")
	assert.NotNil(err, "there should be an error parsing a missing file")

	err = cnfgfile.Unmarshal(config)
	assert.NotNil(err, "there should be an error parsing a nil file")
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}
	err := cnfgfile.Unmarshal(c, "tests/config.json")
	testUnmarshalValues(a, c, err, "TestUnmarshalJSON")
}

func TestUnmarshalXML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := cnfgfile.Unmarshal(c, "tests/config.xml")
	testUnmarshalValues(a, c, err, "TestUnmarshalXML")
}

func TestUnmarshalYAML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := cnfgfile.Unmarshal(c, "tests/config.yaml")
	testUnmarshalValues(a, c, err, "TestUnmarshalYAML")
}

func TestUnmarshalTOML(t *testing.T) {
	t.Parallel()

	a := assert.New(t)
	c := &testStruct{}

	err := cnfgfile.Unmarshal(c, "tests/config.toml")
	testUnmarshalValues(a, c, err, "TestUnmarshalTOML")
}

// The cnfgfile.Unmarshal() procedure can be used in place of: xml.Unmarshal,
// json.Unmarshal, yaml.Unmarshal and toml.Unmarshal. This procedure also reads
// in the provided file, so you don't need to do any of the io work beforehand.
// Using this procedure in your app allows your consumers to a use a config file
// format of their choosing. Very cool stuff when you consider _that file_ could
// just be a config file for a larger project.
func ExampleUnmarshal() {
	// Recommend adding tags for each type to your struct members. Provide full compatibility.
	type Config struct {
		Interval cnfg.Duration `json:"interval" xml:"interval" toml:"interval" yaml:"interval"`
		Location string        `json:"location" xml:"location" toml:"location" yaml:"location"`
		Provided bool          `json:"provided" xml:"provided" toml:"provided" yaml:"provided"`
	}

	// Create a test file with some test data to unmarshal.
	// YAML is just an example, you can use any supported format.
	yaml := []byte("---\ninterval: 5m\nlocation: Earth\nprovided: true")
	path := "/tmp/path_to_config.yaml"

	err := ioutil.WriteFile(path, yaml, 0o600)
	if err != nil {
		panic(err)
	}

	// Start with an empty config. Or set some defaults beforehand.
	config := &Config{}

	// Simply pass in your config file. If it contains ".yaml" it will be parsed as YAML.
	// Same for ".xml" and ".json". If the file has none of these extensions it is parsed
	// as TOML. Meaning if you name your config "config.conf" it needs ot be TOML formatted.
	err = cnfgfile.Unmarshal(config, path)
	if err != nil {
		panic(err)
	}

	fmt.Printf("interval: %v, location: %v, provided: %v", config.Interval, config.Location, config.Provided)
	// Output: interval: 5m, location: Earth, provided: true
}
