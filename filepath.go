package cnfgfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// Changes contains the optional input parameters for ReadConfigs() to control how a data structure is processed.
type Changes struct {
	// Prefix is the string we check for to see if we should read in a config file.
	// If left blank the default of filepath: will be used.
	Prefix string
	// The maximum amount of data we should read in from a config file.
	// If you don't expect large values, leave this small.
	// If left at 0, the default of 1024 is used.
	MaxSize uint
	// Setting NoTrim to true will skip strings.TrimSpace on the read file contents.
	NoTrim bool
	// Name is prefixed to element names when an error is returned.
	// The default name is "Config" if this is omitted.
	Name string
}

// Changes Defaults.
const (
	DefaultPrefix  = "filepath:"
	DefaultMaxSize = uint(1024)
	DefaultName    = "Config"
)

// ReadConfigs parses a data structure and searches for strings. It is fully recursive, and will find strings
// in slices, embedded structs, maps and pointers. If the found string has a defined prefix (filepath: by default),
// then the provided filepath is opened, read, and the contents are saved into the string. Replacing the filepath
// that it once was. This allows you to define a Config struct, and your users can store secrets (or other strings)
// in separate files. After you read in the base config data, pass a pointer to your config struct to this function,
// and it will automatically go to work filling in any extra external config data.
func ReadConfigs(data interface{}, changes *Changes) error {
	if rd := reflect.TypeOf(data); rd.Kind() != reflect.Ptr || rd.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("ReadConfigs: %w", ErrNotPtr)
	}

	if changes == nil {
		changes = &Changes{}
	}

	if changes.MaxSize == 0 {
		changes.MaxSize = DefaultMaxSize
	}

	if changes.Prefix == "" {
		changes.Prefix = DefaultPrefix
	}

	if changes.Name == "" {
		changes.Name = DefaultName
	}

	return changes.parseStruct(reflect.ValueOf(data).Elem(), changes.Name)
}

func (c *Changes) parseStruct(rv reflect.Value, name string) error {
	for i := rv.NumField() - 1; i >= 0; i-- {
		err := c.parseElement(rv.Field(i), name+"."+rv.Type().Field(i).Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Changes) parseMap(field reflect.Value, name string) error {
	for _, key := range field.MapKeys() {
		value := reflect.Indirect(reflect.New(field.MapIndex(key).Type()))
		value.Set(field.MapIndex(key))

		if err := c.parseElement(value, fmt.Sprint(name, key)); err != nil {
			return err
		}

		field.SetMapIndex(key, value)
	}

	return nil
}

func (c *Changes) parseSlice(field reflect.Value, name string) error {
	for i := field.Len() - 1; i >= 0; i-- {
		if err := c.parseElement(field.Index(i), fmt.Sprint(name, i)); err != nil {
			return err
		}
	}

	return nil
}

func (c *Changes) parseElement(field reflect.Value, name string) error {
	switch kind := field.Kind(); kind {
	case reflect.String:
		return c.parseString(field, name)
	case reflect.Struct:
		return c.parseStruct(field, name)
	case reflect.Pointer, reflect.Interface:
		return c.parseElement(field.Elem(), name)
	case reflect.Slice, reflect.Array:
		return c.parseSlice(field, name)
	case reflect.Map:
		return c.parseMap(field, name)
	default:
		return nil
	}
}

func (c *Changes) parseString(field reflect.Value, name string) error {
	value := field.String()
	if !strings.HasPrefix(value, c.Prefix) {
		return nil
	}

	data, err := readFile(strings.TrimPrefix(value, c.Prefix), c.MaxSize)
	if err != nil {
		return fmt.Errorf("element failure: %s: %w", name, err)
	}

	if c.NoTrim {
		field.SetString(data)
	} else {
		field.SetString(strings.TrimSpace(data))
	}

	return nil
}

func readFile(filePath string, maxSize uint) (string, error) {
	fOpen, err := os.OpenFile(strings.TrimSpace(filePath), os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer fOpen.Close()

	data := make([]byte, maxSize)

	size, err := fOpen.Read(data)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(data[:size]), nil
}
