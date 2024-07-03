package cnfgfile

import (
	"encoding"
	"encoding/json"
	"fmt"
	"time"
)

/*** This code also exists in the golift.io/cnfg package. It is identical. ***/

// Duration is useful if you need to load a time Duration from a config file into
// your application. Use the config.Duration type to support automatic unmarshal
// from all sources.
type Duration struct{ time.Duration }

// UnmarshalText parses a duration type from a config file. This method works
// with the Duration type to allow unmarshaling of durations from files and
// env variables in the same struct. You won't generally call this directly.
func (d *Duration) UnmarshalText(b []byte) error {
	dur, err := time.ParseDuration(string(b))
	if err != nil {
		return fmt.Errorf("parsing duration '%s': %w", b, err)
	}

	d.Duration = dur

	return nil
}

// MarshalText returns the string representation of a Duration. ie. 1m32s.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// MarshalJSON returns the string representation of a Duration for JSON. ie. "1m32s".
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Duration.String() + `"`), nil
}

// String returns a Duration as string without trailing zero units.
func (d Duration) String() string {
	dur := d.Duration.String()
	if len(dur) > 3 && dur[len(dur)-3:] == "m0s" {
		dur = dur[:len(dur)-2]
	}

	if len(dur) > 3 && dur[len(dur)-3:] == "h0m" {
		dur = dur[:len(dur)-2]
	}

	return dur
}

// Make sure our struct satisfies the interface it's for.
var (
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ encoding.TextMarshaler   = (*Duration)(nil)
	_ json.Marshaler           = (*Duration)(nil)
	_ fmt.Stringer             = (*Duration)(nil)
)
