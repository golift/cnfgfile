// Package cnfgfile provides two distinct features.
// 1. A shorthand procedure to unmarshal any config file(s).
// You can put your configuration into any file format: XML, YAML, JSON, TOML.
// You can pass in more than one config file to unmarshal a hierarchy of configs.
// 2. Provides a way to read in config settings from their own files. This is most
// useful for keeping secrets in their own files. You can pass a struct into
// ReadConfigs and it will check every string for a filepath: prefix. When it finds
// a string with that prefix, it reads in the filepath that follows, and re-sets the
// string to the value found in the file.
package cnfgfile

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	toml "github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v3"
)

// Errors this library may produce.
var (
	ErrPanic  = errors.New("bug in the golift.io/cnfgfile package; caught panic")
	ErrNoFile = errors.New("must provide at least 1 file to unmarshal")
	ErrNotPtr = errors.New("ReadConfigs: must provide a pointer to data structure that can be modified")
)

// Unmarshal parses a configuration file (of any format) into a config struct.
// This is a shorthand method for calling Unmarshal against the json, xml, yaml
// or toml packages. If the file name contains an appropriate suffix it is
// unmarshaled with the corresponding package. If the suffix is missing, TOML
// is assumed. Works with multiple files, so you can have stacked configurations.
// Will detect (and decompress) a file that is gzip or bzip2 compressed.
func Unmarshal(config interface{}, configFile ...string) error {
	if len(configFile) == 0 {
		return ErrNoFile
	}

	for _, fileName := range configFile {
		fileOpen, err := os.Open(fileName)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", fileName, err)
		}
		defer fileOpen.Close()

		fileReader, err := deCompress(fileOpen)

		switch lowerName := strings.ToLower(fileName); {
		case err != nil:
			return err
		case strings.Contains(lowerName, ".json"):
			err = json.NewDecoder(fileReader).Decode(config)
		case strings.Contains(lowerName, ".xml"):
			err = xml.NewDecoder(fileReader).Decode(config)
		case strings.Contains(lowerName, ".yaml"), strings.Contains(lowerName, ".yml"):
			err = yaml.NewDecoder(fileReader).Decode(config)
		default:
			_, err = toml.NewDecoder(fileReader).Decode(config)
		}

		if err != nil {
			return fmt.Errorf("unmarshaling file %s: %w", fileName, err)
		}
	}

	return nil
}

func deCompress(fileReader *os.File) (io.Reader, error) {
	buff := make([]byte, 512) //nolint:mnd
	if _, err := fileReader.Read(buff); err != nil {
		return nil, fmt.Errorf("reading file %s: %w", fileReader.Name(), err)
	}

	if _, err := fileReader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seeking file start %s: %w", fileReader.Name(), err)
	}

	switch {
	case http.DetectContentType(buff) == "application/x-gzip":
		gz, err := gzip.NewReader(fileReader)
		if err != nil {
			return nil, fmt.Errorf("file detected as gz, decompress failed: %w", err)
		}

		return gz, nil
	case strings.HasPrefix(string(buff), "\x42\x5a\x68"):
		return bzip2.NewReader(fileReader), nil
	default:
		return fileReader, nil
	}
}
