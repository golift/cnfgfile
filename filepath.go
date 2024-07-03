package cnfgfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// Opts contains the optional input parameters for ReadConfigs() to control how a data structure is processed.
type Opts struct {
	// Prefix is the string we check for to see if we should read in a config file.
	// If left blank the default of filepath: will be used.
	Prefix string
	// The maximum amount of bytes that should read in from an external config file.
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
// and it will automatically go to work filling in any extra external config data. Opts may be nil, uses defaults.
func ReadConfigs(input interface{}, opts *Opts) error {
	data := reflect.TypeOf(input)
	if data.Kind() != reflect.Ptr || data.Elem().Kind() != reflect.Struct {
		return ErrNotPtr
	}

	if opts == nil {
		opts = &Opts{}
	}

	if opts.MaxSize == 0 {
		opts.MaxSize = DefaultMaxSize
	}

	if opts.Prefix == "" {
		opts.Prefix = DefaultPrefix
	}

	if opts.Name == "" {
		opts.Name = DefaultName
	}

	return opts.parseStruct(reflect.ValueOf(input).Elem(), opts.Name)
}

func (o *Opts) parseStruct(field reflect.Value, name string) error {
	for i := field.NumField() - 1; i >= 0; i-- {
		name := name + "." + field.Type().Field(i).Name // name is overloaded here.
		if err := o.parseElement(field.Field(i), name); err != nil {
			return err
		}
	}

	return nil
}

func (o *Opts) parseMap(field reflect.Value, name string) error {
	for _, key := range field.MapKeys() {
		// Copy the map field.
		fieldCopy := reflect.Indirect(reflect.New(field.MapIndex(key).Type()))
		// Set the copy's value to the value of the original.
		fieldCopy.Set(field.MapIndex(key))

		// Parse the copy, because map values cannot be .Set() directly.
		name := fmt.Sprint(name, "[", key, "]") // name is overloaded here.
		if err := o.parseElement(fieldCopy, name); err != nil {
			return err
		}

		// Update the map index with the possibly-modified copy that got parsed.
		field.SetMapIndex(key, fieldCopy)
	}

	return nil
}

func (o *Opts) parseSlice(field reflect.Value, name string) error {
	for idx := field.Len() - 1; idx >= 0; idx-- {
		name := fmt.Sprint(name, "[", idx+1, "/", field.Len(), "]") // name is overloaded here.
		if err := o.parseElement(field.Index(idx), name); err != nil {
			return err
		}
	}

	return nil
}

// parseElement processes any supported element type.
func (o *Opts) parseElement(field reflect.Value, name string) error {
	switch kind := field.Kind(); kind {
	case reflect.String:
		return o.parseString(field, name)
	case reflect.Struct:
		return o.parseStruct(field, name)
	case reflect.Pointer, reflect.Interface:
		return o.parseElement(field.Elem(), name)
	case reflect.Slice, reflect.Array:
		return o.parseSlice(field, name)
	case reflect.Map:
		return o.parseMap(field, name)
	default:
		return nil
	}
}

func (o *Opts) parseString(field reflect.Value, name string) error {
	value := field.String()
	if !strings.HasPrefix(value, o.Prefix) {
		return nil
	}

	data, err := readFile(strings.TrimPrefix(value, o.Prefix), o.MaxSize)
	if err != nil {
		return fmt.Errorf("element failure: %s: %w", name, err)
	}

	if o.NoTrim {
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
