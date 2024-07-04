package cnfgfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// Opts contains the optional input parameters for Parse() to control how a data structure is processed.
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
	// Name is prefixed to element names. You will find the derived name in errors, and in the map output.
	// The default name is "Config" if this is omitted.
	Name   string
	output map[string]string
}

// Parse(Opts) Defaults.
const (
	DefaultPrefix  = "filepath:"
	DefaultMaxSize = uint(1024)
	DefaultName    = "Config"
)

// ParseError is returned when there's an error reading a string-parsed file.
type ParseError struct {
	// name of the failed element
	Element string
	// name of the failed file
	FilePath string
	// error returned reading the file
	Inner error
}

// Error satisfies the standard Go library error interface.
// We do not print the filepath because it's (always?) included in the Inner error.
func (p *ParseError) Error() string {
	const prefix = "element failure"

	return prefix + ": " + p.Element + ": " + p.Inner.Error()
}

// Unwrap is used to make the custom error work with errors.Is and errors.As.
func (p *ParseError) Unwrap() error {
	return p.Inner // Return the wrapped error.
}

// Parse parses a data structure and searches for strings. It is fully recursive, and will find strings
// in slices, embedded structs, maps and pointers. If the found string has a defined prefix (filepath: by default),
// then the provided filepath is opened, read, and the contents are saved into the string. Replacing the filepath
// that it once was. This allows you to define a Config struct, and your users can store secrets (or other strings)
// in separate files. After you read in the base config data, pass a pointer to your config struct to this function,
// and it will automatically go to work filling in any extra external config data. Opts may be nil, uses defaults.
// The output map is a map of Config.Item => filepath. Use this to see what files were read-in for each config path.
// If there is an element failure, the failed element and all prior parsed elements will be present in the map.
// Unwrap errors into a ParseError to get the failed file name and a derived name of the element it was found in.
func Parse(input interface{}, opts *Opts) (map[string]string, error) {
	data := reflect.TypeOf(input)
	if data.Kind() != reflect.Ptr || data.Elem().Kind() != reflect.Struct {
		return nil, ErrNotPtr
	}

	opts = getOpts(opts)
	// parse the input struct and return the outmap.
	return opts.output, opts.parseStruct(reflect.ValueOf(input).Elem(), opts.Name)
}

// opts is an optional input, but required in this package.
func getOpts(opts *Opts) *Opts {
	if opts == nil {
		return &Opts{
			Prefix:  DefaultPrefix,
			MaxSize: DefaultMaxSize,
			Name:    DefaultName,
			output:  make(map[string]string),
		}
	}

	// Set defaults for omitted values.
	if opts.MaxSize == 0 {
		opts.MaxSize = DefaultMaxSize
	}

	if opts.Prefix == "" {
		opts.Prefix = DefaultPrefix
	}

	if opts.Name == "" {
		opts.Name = DefaultName
	}

	// return a copy to make the map thread safe.
	return &Opts{
		Prefix:  opts.Prefix,
		MaxSize: opts.MaxSize,
		NoTrim:  opts.NoTrim,
		Name:    opts.Name,
		output:  make(map[string]string),
	}
}

// If you pass in a non-struct to this function, you'll experience a panic.
func (o *Opts) parseStruct(elem reflect.Value, name string) error {
	for i := elem.NumField() - 1; i >= 0; i-- {
		name := name + "." + elem.Type().Field(i).Name // name is overloaded here.
		if err := o.parseElement(elem.Field(i), name); err != nil {
			return err
		}
	}

	return nil
}

// If you pass in a non-map to this function, you'll experience a panic.
func (o *Opts) parseMap(elem reflect.Value, name string) error {
	for _, key := range elem.MapKeys() {
		// Copy the map field.
		fieldCopy := reflect.Indirect(reflect.New(elem.MapIndex(key).Type()))
		// Set the copy's value to the value of the original.
		fieldCopy.Set(elem.MapIndex(key))

		// Parse the copy, because map values cannot be .Set() directly.
		name := fmt.Sprint(name, "[", key, "]") // name is overloaded here.
		if err := o.parseElement(fieldCopy, name); err != nil {
			return err
		}

		// Update the map index with the possibly-modified copy that got parsed.
		elem.SetMapIndex(key, fieldCopy)
	}

	return nil
}

func (o *Opts) parseSlice(slice reflect.Value, name string) error {
	for idx := slice.Len() - 1; idx >= 0; idx-- {
		name := fmt.Sprint(name, "[", idx+1, "/", slice.Len(), "]") // name is overloaded here.
		if err := o.parseElement(slice.Index(idx), name); err != nil {
			return err
		}
	}

	return nil
}

// parseElement processes any supported element type.
func (o *Opts) parseElement(elem reflect.Value, name string) error {
	switch kind := elem.Kind(); kind { //nolint:exhaustive
	case reflect.String:
		return o.parseString(elem, name)
	case reflect.Struct:
		return o.parseStruct(elem, name)
	case reflect.Pointer, reflect.Interface:
		return o.parseElement(elem.Elem(), name)
	case reflect.Slice, reflect.Array:
		return o.parseSlice(elem, name)
	case reflect.Map:
		return o.parseMap(elem, name)
	default:
		return nil
	}
}

func (o *Opts) parseString(elem reflect.Value, name string) error {
	value := elem.String()
	if !strings.HasPrefix(value, o.Prefix) {
		return nil
	}

	// Save this parsed file to the output map. Remove the prefix and any enclosing whitespace.
	o.output[name] = strings.TrimSpace(strings.TrimPrefix(value, o.Prefix))

	data, err := readFile(o.output[name], o.MaxSize)
	if err != nil {
		return &ParseError{
			Element:  name,
			FilePath: o.output[name],
			Inner:    err,
		}
	}

	if o.NoTrim {
		elem.SetString(data)
	} else {
		elem.SetString(strings.TrimSpace(data))
	}

	return nil
}

func readFile(filePath string, maxSize uint) (string, error) {
	fOpen, err := os.OpenFile(filePath, os.O_RDONLY, 0)
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
