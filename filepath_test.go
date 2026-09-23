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
	invisible    string
}

type dataStruct struct {
	TestStruct
	String

	Name      string
	Address   string
	Interface any
	Sliceface []any
	Number    int
	Embed     struct {
		EmbedName    string
		EmbedAddress string
		EmbedNumber  int
	}
	Named   *TestStruct
	Map     map[string]string
	MapI    map[int]string
	LulWut  map[any][]*TestStruct
	Strings []string
	Structs []TestStruct
	Ptructs []*TestStruct
	Etring  String
	StrPtr  *String
}

type String string

func TestParse(t *testing.T) {
	t.Parallel()

	file := makeTextFile(t)
	defer os.Remove(file)

	data := testData(t, file)
	testString := strings.TrimSuffix(testString, "\n")

	output, err := cnfgfile.Parse(&data, nil)
	require.NoError(t, err, "got an unexpected error")
	assert.Equal(t, testString, data.Address)
	assert.Equal(t, testString, data.Embed.EmbedAddress)
	assert.Equal(t, testString, data.Named.EmbedAddress)
	assert.Equal(t, testString, data.EmbedAddress)
	assert.Equal(t, cnfgfile.DefaultPrefix+file, data.invisible, "do not modify non-exported members")
	assert.Equal(t, testString, data.Strings[1])
	assert.Equal(t, testString, data.Structs[0].EmbedAddress)
	assert.Equal(t, testString, data.Ptructs[0].EmbedAddress)
	assert.EqualValues(t, testString, data.String)
	assert.EqualValues(t, testString, data.Etring)
	assert.EqualValues(t, testString, *data.StrPtr)

	assert.Equal(t, testString, data.Map["map_string"])
	assert.Equal(t, "data stuff", data.Map["map2_string"])
	assert.Equal(t, testString, data.MapI[2], "an unexpected change was made to a string")
	assert.Equal(t, "data stuff", data.MapI[5], "an unexpected change was made to a string")
	assert.Len(t, output, 12, "12 items have filepath: in them and should be returned")

	data.Name = "super:" + file
	output, err = cnfgfile.Parse(&data, &cnfgfile.Opts{Prefix: "super:", MaxSize: 8, NoTrim: true})
	require.NoError(t, err)
	assert.Equal(t, testString[:8], data.Name, "opts.MaxSize doesn't seem to be working")
	assert.Len(t, output, 1, "only 1 item should be in the output map")
	assert.Equal(t, output["Config.Name"], file, "the parsed file is not in the config map")
}

func TestParseErrors(t *testing.T) {
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
	// Without a max depth limit, this recursive struct pointer will hit the
	// 10000 thread limit, or use all available system memory before crashing.
	data.StarStruck = &data.TestStruct

	_, err := cnfgfile.Parse(data, opts)
	require.ErrorIs(t, err, cnfgfile.ErrNotPtr)

	data.Name = "super:/no_file"
	// This test:
	// makes sure the correct opts.Prefix is used.
	// makes sure the proper opts.Name is used.
	// makes sure a missing file returns a useful error.
	_, err = cnfgfile.Parse(&data, opts)
	require.ErrorContains(t, err,
		"element failure: MyThing.Name: opening file: open /no_file:",
		"this may indicate the wrong prefix or name is being used")

	data.Name = ""
	data.Map["MAPKEY"] = "super:/no_file"
	_, err = cnfgfile.Parse(&data, opts)
	require.ErrorContains(t, err,
		"element failure: MyThing.Map[MAPKEY]: opening file: open /no_file:",
		"this may indicate the wrong prefix or name is being used")

	delete(data.Map, "MAPKEY")
	data.LulWut = map[any][]*TestStruct{"some_key": {nil, {EmbedName: "super:/no_file"}, nil}}
	_, err = cnfgfile.Parse(&data, opts)
	require.ErrorContains(t, err,
		"element failure: MyThing.LulWut[some_key][2/3].EmbedName: opening file: open /no_file:",
		"this test fails is the member names are not concatenated properly")

	data.LulWut = map[any][]*TestStruct{
		String("flop"): {nil, {StarStruck: &TestStruct{MemberName: []String{"super:/no_file", ""}}}},
	}
	_, err = cnfgfile.Parse(&data, opts)
	require.ErrorContains(t, err,
		"element failure: MyThing.LulWut[flop][2/2].StarStruck.MemberName[1/2]: opening file: open /no_file:",
		"this test fails is the member names are not concatenated properly")
}

// testData returns a test struct filled with filepaths.
// We test strings, structs, maps, slices, pointers...
func testData(t *testing.T, file string) dataStruct {
	t.Helper()

	str := String(cnfgfile.DefaultPrefix + file)

	return dataStruct{
		Name:      "me",
		Address:   cnfgfile.DefaultPrefix + file,
		Interface: cnfgfile.DefaultPrefix + file,
		Sliceface: []any{nil, cnfgfile.DefaultPrefix + file},
		Embed: struct {
			EmbedName    string
			EmbedAddress string
			EmbedNumber  int
		}{
			EmbedAddress: cnfgfile.DefaultPrefix + file,
		},
		TestStruct: TestStruct{
			invisible:    cnfgfile.DefaultPrefix + file,
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

	fOpen, err := os.CreateTemp(t.TempDir(), "cnfgfile_*_test")
	require.NoError(t, err, "unable to create temporary file")

	defer fOpen.Close()

	size, err := fOpen.WriteString(testString)
	require.NoError(t, err, "unable to write temporary file data")
	assert.Len(t, testString, size, "wrong data size writing temporary file")

	return fOpen.Name()
}
