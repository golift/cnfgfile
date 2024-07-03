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

type TestStruct struct {
	EmbedName    string
	EmbedAddress string
	EmbedNumber  int
	MemberName   []String
	StarStruck   *TestStruct
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
	TestStruct
	Named   *TestStruct
	Map     map[string]string
	MapI    map[int]string
	LulWut  map[interface{}][]*TestStruct
	Strings []string
	Structs []TestStruct
	Ptructs []*TestStruct
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
	assert.EqualValues(t, testString, data.TestStruct.EmbedAddress)
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
	data.LulWut = map[interface{}][]*TestStruct{"some_key": {nil, {EmbedName: "super:/no_file"}, nil}}
	require.ErrorContains(t, cnfgfile.ReadConfigs(&data, opts),
		"element failure: MyThing.LulWut[some_key][2/3].EmbedName: opening file: open /no_file:",
		"this test fails is the member names are not concatenated properly")

	data.LulWut = map[interface{}][]*TestStruct{
		String("flop"): {nil, {StarStruck: &TestStruct{MemberName: []String{"super:/no_file", ""}}}},
	}
	require.ErrorContains(t, cnfgfile.ReadConfigs(&data, opts),
		"element failure: MyThing.LulWut[flop][2/2].StarStruck.MemberName[1/2]: opening file: open /no_file:",
		"this test fails is the member names are not concatenated properly")
}

// testData returns a test struct filled with filepaths.
// We test strings, structs, maps, slices, pointers...
func testData(t *testing.T, file string) dataStruct {
	t.Helper()

	str := String(cnfgfile.DefaultPrefix + file)

	return dataStruct{
		Name:    "me",
		Address: cnfgfile.DefaultPrefix + file,
		Embed: struct {
			EmbedName    string
			EmbedAddress string
			EmbedNumber  int
		}{
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		},
		TestStruct: TestStruct{
			EmbedName:    "me2",
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		},
		Named: &TestStruct{
			EmbedName:    "me3",
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		},
		Map: map[string]string{
			"map_string":  cnfgfile.DefaultPrefix + file,
			"map2_string": "data stuff",
		},
		MapI: map[int]string{
			2: cnfgfile.DefaultPrefix + file,
			5: "data stuff",
		},
		Strings: []string{"foo", cnfgfile.DefaultPrefix + file},
		Structs: []TestStruct{{
			EmbedName:    "me4",
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		}},
		Ptructs: []*TestStruct{{
			EmbedName:    "me5",
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		}},
		String: String(cnfgfile.DefaultPrefix + file),
		Etring: String(cnfgfile.DefaultPrefix + file),
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
