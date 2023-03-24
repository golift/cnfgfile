// Package cnfgfile provides a shorthand procedure to unmarshal any config file(s).
// You can put your configuration into any file format: XML, YAML, JSON, TOML.
// You can pass in more than one config file to unmarshal a hierarchy of configs.
// Works well with parent cnfg package. Call this package or cnfg in either order.
// The former overrides the latter.
package cnfgfile

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	toml "github.com/BurntSushi/toml"
	yaml "gopkg.in/yaml.v3"
)

var ErrNoFile = fmt.Errorf("must provide at least 1 file to unmarshal")

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
		case strings.Contains(lowerName, ".yaml"):
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
	buff := make([]byte, 512) //nolint:gomnd
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
