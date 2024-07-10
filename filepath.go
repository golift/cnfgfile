package cnfgfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
)

// Opts contains the optional input parameters for Parse() to control how a data structure is processed.
type Opts struct {
	// Name is prefixed to element names. You will find the derived name in errors, and in the map output.
	// The default name is "Config" if this is omitted.
	Name string
	// Prefix is the string we check for to see if we should read in an external config file.
	// If a string contains this prefix, the data that follows is treated as a file path.
	// The data is read in from the file and replaces the string.
	// If left blank the default of filepath: will be used.
	Prefix string
	// Setting NoTrim to true will skip TrimSpace on the data read from the external config file.
	// If this is true, and the file ends with a newline, it will be included in the updated string value.
	NoTrim bool
	// MaxSize is the maximum amount of bytes that are read in from an external config file.
	// If you don't expect large values, leave this small. If left at 0, the default of 1024 is used.
	MaxSize uint
	// MaxDepths controls how deep into nested structs this package will recurse. If left unchecked, a
	// recursive pointer may use all your memory and crash, so a maximum is required. If left at 0, the
	// default of 200 is set.
	MaxDepth uint
}

// Parse(Opts) Defaults.
const (
	DefaultPrefix   = "filepath:"
	DefaultMaxSize  = uint(1024)
	DefaultName     = "Config"
	DefaultMaxDepth = uint(200)
)

// ElemError is returned as an error interface when there's an error reading a string-parsed file.
// Use errors.As() to make this data available in your application.
type ElemError struct {
	// Name of the failed element.
	Name string
	// File name (path) of the failed file.
	File string
	// Inner error returned reading the file.
	Inner error
}

// Error satisfies the standard Go library error interface.
// We do not print the filepath because it's (always?) included in the Inner error.
func (p *ElemError) Error() string {
	const prefix = "element failure"

	return prefix + ": " + p.Name + ": " + p.Inner.Error()
}

// Unwrap is used to make the custom error work with errors.Is and errors.As.
func (p *ElemError) Unwrap() error {
	return p.Inner // Return the wrapped error.
}

// Parse parses a data structure from a pointer, and searches for strings. It is fully recursive, and finds strings
// in slices, embedded structs, maps and pointers. If the found string has a defined prefix (filepath: by default),
// then the provided filepath is opened, read, and the contents are saved into the string. Replacing the filepath
// that it once was. This allows you to define a Config struct, and your users can store secrets (or other strings)
// in separate files. After you read in the base config data, pass a pointer to your config struct to this function,
// and it will automatically go to work filling in any extra external config data. Opts may be nil, uses defaults.
// The output map is a map of Config.Item => filepath. Use this to see what files were read-in for each config path.
// If there is an element failure, the failed element and all prior parsed elements will be present in the map.
// Unwrap errors into a ElemError type to get the failed file name and a derived name of the element it was found in.
func Parse(ptr interface{}, opts *Opts) (_ map[string]string, err error) {
	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return nil, ErrNotPtr
	}

	parser := opts.newParser()

	defer func() {
		if r := recover(); r != nil {
			err = &ElemError{ // Update the returned error.
				Name:  parser.name,
				File:  "",
				Inner: fmt.Errorf("%w: %v\n%s", ErrPanic, r, string(debug.Stack())),
			}
		}
	}()

	// Parse the input element and return the output map.
	return parser.Output, parser.Parse(reflect.ValueOf(ptr), parser.Name)
}

// parser is used for internal methods.
type parser struct {
	// Opts is the input parameters.
	Opts
	// Output is where we store the map of element => filepath that gets returned to the caller.
	Output map[string]string
	// depth is the current depth while parsing.
	depth uint
	// name is the current element being parsed. Returned in an error in case of panic.
	name string
}

// newParser returns a parser with attached Opts. Sets defaults for any omitted values.
func (o *Opts) newParser() *parser {
	output := &parser{
		Output: make(map[string]string),
		Opts: Opts{ // Create a copy to make changes thread safe.
			Name:     DefaultName,
			Prefix:   DefaultPrefix,
			NoTrim:   false,
			MaxSize:  DefaultMaxSize,
			MaxDepth: DefaultMaxDepth,
		},
		depth: 0,
		name:  DefaultName,
	}

	if o == nil {
		return output // Nothing to copy, return defaults.
	}

	output.NoTrim = o.NoTrim
	output.name = o.Name
	// Copy values, and set defaults for omitted values.
	if output.Name = o.Name; output.Name == "" {
		output.Name = DefaultName
	}

	if output.Prefix = o.Prefix; output.Prefix == "" {
		output.Prefix = DefaultPrefix
	}

	if output.MaxSize = o.MaxSize; output.MaxSize == 0 {
		output.MaxSize = DefaultMaxSize
	}

	if output.MaxDepth = o.MaxDepth; output.MaxDepth == 0 {
		output.MaxDepth = DefaultMaxDepth
	}

	return output
}

// Parse processes any supported element type, and it gets called recursively a lot in this package.
func (p *parser) Parse(element reflect.Value, name string) error {
	p.depth++
	defer func() { p.depth-- }()

	if p.depth >= p.MaxDepth {
		// return fmt.Errorf("max depth [%d/%d]: %v", p.Depth, p.MaxDepth, name)
		return nil
	}

	if parse := p.parseFunc(element); parse != nil {
		return parse(element, name)
	}

	return nil // Unsupported type.
}

// parseFunc contains all the supported kinds and their corresponding parse method.
// Returns nil if the provided kind is not supported.
func (p *parser) parseFunc(elem reflect.Value) func(reflect.Value, string) error {
	return map[reflect.Kind]func(reflect.Value, string) error{
		reflect.Interface: p.parsePointer,
		reflect.Pointer:   p.parsePointer,
		reflect.String:    p.parseString,
		reflect.Struct:    p.parseStruct,
		reflect.Slice:     p.parseSlice,
		reflect.Array:     p.parseSlice,
		reflect.Map:       p.parseMap,
	}[elem.Kind()]
}

// parsePointer allows dereferencing pointers and interfaces before passing them to the element parser.
func (p *parser) parsePointer(elem reflect.Value, name string) error {
	if elem.IsNil() {
		return nil
	}

	return p.Parse(elem.Elem(), name) // We could suffix the name here.
}

// If you pass in a non-struct to this function, you'll experience a panic.
func (p *parser) parseStruct(elem reflect.Value, name string) error {
	for _, field := range reflect.VisibleFields(elem.Type()) {
		p.name = name + "." + field.Name

		member, err := elem.FieldByIndexErr(field.Index)
		if !field.IsExported() || err != nil {
			continue // Only mess with visible, exported non-nil struct members.
		}

		if err := p.Parse(member, p.name); err != nil {
			return err
		}
	}

	return nil
}

// If you pass in a non-map to this function, you'll experience a panic.
func (p *parser) parseMap(elem reflect.Value, name string) error {
	keys := elem.MapKeys()
	if len(keys) == 0 || p.parseFunc(elem.MapIndex(keys[0])) == nil {
		return nil // Avoid traversing map types that don't contain strings.
	}

	for _, key := range keys {
		// Copy the map field type, using this ridiculous reflect magic.
		elemCopy := reflect.Indirect(reflect.New(elem.MapIndex(key).Type()))
		// Set the copy's value to the value of the original.
		elemCopy.Set(elem.MapIndex(key))

		// Parse the copy, because map values cannot be .Set() directly.
		p.name = fmt.Sprint(name, "[", key, "]")
		if err := p.Parse(elemCopy, p.name); err != nil {
			return err
		}

		// Update the map index with the possibly-modified copy that got parsed.
		elem.SetMapIndex(key, elemCopy)
	}

	return nil
}

// parseSlice traverses all slice elements if the slice kind is supported.
func (p *parser) parseSlice(slice reflect.Value, name string) error {
	length := slice.Len()
	if length == 0 || p.parseFunc(slice.Index(0)) == nil {
		return nil // Avoid traversing byte slices and other things that don't contain strings.
	}

	for idx := length - 1; idx >= 0; idx-- {
		p.name = fmt.Sprintf("%s[%d/%d]", name, idx+1, length)
		if err := p.Parse(slice.Index(idx), p.name); err != nil {
			return err
		}
	}

	return nil
}

// This parse function is non-recursive. The buck stops here, so to speak.
// If the string has the correct prefix, and can be set, read the file and set it!
func (p *parser) parseString(elem reflect.Value, name string) error {
	value := elem.String()
	if !elem.CanSet() || !strings.HasPrefix(value, p.Prefix) {
		return nil
	}

	// Save this parsed file to the output map. Remove the prefix and any enclosing whitespace.
	p.Output[name] = strings.TrimSpace(strings.TrimPrefix(value, p.Prefix))
	// Read in the file contents.
	fileContent, err := p.readFile(p.Output[name])
	if err != nil {
		return &ElemError{ // Warp the error with our custom type.
			Name:  name,
			File:  p.Output[name],
			Inner: err,
		}
	}

	elem.SetString(fileContent) // Update the string element's value with the file contents.

	return nil
}

// Read and return a file's contents according to requested byte size and trim or not.
func (p *parser) readFile(filePath string) (string, error) {
	fOpen, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer fOpen.Close()

	// This is how .Read() works, it will return this many bytes (or less).
	fileContent := make([]byte, p.MaxSize)
	// size is the amount (count) of data (bytes) read.
	size, err := fOpen.Read(fileContent)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("reading file: %w", err)
	}

	if p.NoTrim { // Leave any newlines or other enclosing whitespace.
		return string(fileContent[:size]), nil
	}
	// The [:size] trims off the extra junk from the empty byte slice.
	return string(bytes.TrimSpace(fileContent[:size])), nil
}
