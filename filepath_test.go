package cnfgfile_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golift.io/cnfgfile"
)

const testString = "hi, this is a string\n"

type Struct struct {
	EmbedName    string
	EmbedAddress string
	EmbedNumber  int
}

type dataStruct struct {
	Name    string
	Address string
	Number  int
	Embed   struct {
		EmbedName    string
		EmbedAddress string
		EmbedNumber  int
	}
	Struct
	Named   *Struct
	Map     map[string]string
	MapI    map[int]string
	LulWut  map[interface{}][]*Struct
	Strings []string
	Structs []Struct
	Ptructs []*Struct
	Etring  String
	String
	StrPtr *String
}

type String string

func TestReadConfigs(t *testing.T) {
	t.Parallel()

	file := makeTextFile(t)
	defer os.Remove(file)

	data := testData(t, file)
	testString := strings.TrimSuffix(testString, "\n")

	require.NoError(t, cnfgfile.ReadConfigs(&data, nil), "got an unexpected error")
	assert.EqualValues(t, testString, data.Address)
	assert.EqualValues(t, testString, data.Embed.EmbedAddress)
	assert.EqualValues(t, testString, data.Named.EmbedAddress)
	assert.EqualValues(t, testString, data.Struct.EmbedAddress)
	assert.EqualValues(t, testString, data.Strings[1])
	assert.EqualValues(t, testString, data.Structs[0].EmbedAddress)
	assert.EqualValues(t, testString, data.Ptructs[0].EmbedAddress)
	assert.EqualValues(t, testString, data.String)
	assert.EqualValues(t, testString, data.Etring)
	assert.EqualValues(t, testString, *data.StrPtr)

	assert.EqualValues(t, testString, data.Map["map_string"])
	assert.EqualValues(t, "data stuff", data.Map["map2_string"])
	assert.EqualValues(t, testString, data.MapI[2], "an unexpected change was made to a string")
	assert.EqualValues(t, "data stuff", data.MapI[5], "an unexpected change was made to a string")

	data.Name = "super:" + file
	require.NoError(t, cnfgfile.ReadConfigs(&data, &cnfgfile.Opts{Prefix: "super:", MaxSize: 8}))
	assert.Equal(t, testString[:8], data.Name, "opts.MaxSize doesn't seem to be working")
}

func TestReadConfigsErrors(t *testing.T) {
	t.Parallel()

	file := makeTextFile(t)
	defer os.Remove(file)

	data := testData(t, file)
	opts := &cnfgfile.Opts{
		Prefix:  "super:",
		MaxSize: 8,
		NoTrim:  true,
		Name:    "MyThing",
	}

	require.ErrorIs(t, cnfgfile.ReadConfigs(data, opts), cnfgfile.ErrNotPtr)

	data.Name = "super:/no_file"
	// This test:
	// makes sure the correct opts.Prefix is used.
	// makes sure the proper opts.Name is used.
	// makes sure a missing file returns a useful error.
	require.ErrorContains(t, cnfgfile.ReadConfigs(&data, opts),
		"element failure: MyThing.Name: opening file: open /no_file:",
		"this may indicate the wrong prefix or name is being used")

	data.Name = ""
	data.Map["MAPKEY"] = "super:/no_file"
	require.ErrorContains(t, cnfgfile.ReadConfigs(&data, opts),
		"element failure: MyThing.Map[MAPKEY]: opening file: open /no_file:",
		"this may indicate the wrong prefix or name is being used")

	delete(data.Map, "MAPKEY")
	data.LulWut = map[interface{}][]*Struct{"some_key": {nil, {EmbedName: "super:/no_file"}, nil}}
	require.ErrorContains(t, cnfgfile.ReadConfigs(&data, opts),
		"element failure: MyThing.LulWut[some_key][2/3].EmbedName: opening file: open /no_file:",
		"this test fails is the member names are not concatenated properly")
}

// testData returns a test struct filled with filepaths.
// We test strings, structs, maps, slices, pointers...
func testData(t *testing.T, file string) dataStruct {
	t.Helper()

	str := String("filepath:" + file)

	return dataStruct{
		Name:    "me",
		Address: "filepath:" + file,
		Embed: struct {
			EmbedName    string
			EmbedAddress string
			EmbedNumber  int
		}{
			EmbedAddress: "filepath:" + file,
		},
		Struct: Struct{
			EmbedName:    "me2",
			EmbedAddress: "filepath:" + file,
		},
		Named: &Struct{
			EmbedName:    "me3",
			EmbedAddress: "filepath:" + file,
		},
		Map: map[string]string{
			"map_string":  "filepath:" + file,
			"map2_string": "data stuff",
		},
		MapI: map[int]string{
			2: "filepath:" + file,
			5: "data stuff",
		},
		Strings: []string{"foo", "filepath:" + file},
		Structs: []Struct{{
			EmbedName:    "me4",
			EmbedAddress: "filepath:" + file,
		}},
		Ptructs: []*Struct{{
			EmbedName:    "me5",
			EmbedAddress: "filepath:" + file,
		}},
		String: String("filepath:" + file),
		Etring: String("filepath:" + file),
		StrPtr: &str,
	}
}

func makeTextFile(t *testing.T) string {
	t.Helper()

	fOpen, err := os.CreateTemp("", "cnfgfile_*_test")
	require.NoError(t, err, "unable to create temporary file")
	defer fOpen.Close()

	size, err := fOpen.WriteString(testString)
	require.NoError(t, err, "unable to write temporary file data")
	assert.Len(t, testString, size, "wrong data size writing temporary file")

	return fOpen.Name()
}
