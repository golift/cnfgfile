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
	Strings []string
	Structs []Struct
	Ptructs []*Struct
}

func TestReadConfigs(t *testing.T) {
	t.Parallel()

	file := makeTextFile(t)
	defer os.Remove(file)

	data := dataStruct{
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
	}
	testString := strings.TrimSuffix(testString, "\n")

	require.NoError(t, cnfgfile.ReadConfigs(&data, nil), "got an unexpected error")
	assert.EqualValues(t, testString, data.Address)
	assert.EqualValues(t, testString, data.Embed.EmbedAddress)
	assert.EqualValues(t, testString, data.Named.EmbedAddress)
	assert.EqualValues(t, testString, data.Struct.EmbedAddress)
	assert.EqualValues(t, testString, data.Strings[1])
	assert.EqualValues(t, testString, data.Structs[0].EmbedAddress)
	assert.EqualValues(t, testString, data.Ptructs[0].EmbedAddress)
	assert.EqualValues(t, testString, data.Map["map_string"])
	assert.EqualValues(t, "data stuff", data.Map["map2_string"])
	assert.EqualValues(t, testString, data.MapI[2], "an unexpected change was made to a string")
	assert.EqualValues(t, "data stuff", data.MapI[5], "an unexpected change was made to a string")
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
