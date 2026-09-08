package client

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// FlexInt is an int64 that tolerates the i-doit API habit of encoding numeric
// identifiers sometimes as JSON numbers and sometimes as JSON strings (and
// occasionally as an empty string or null).
type FlexInt int64

func (f *FlexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}

	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*f = 0
			return nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			// Some fields arrive as "12.00"; fall back to float parsing.
			fv, ferr := strconv.ParseFloat(s, 64)
			if ferr != nil {
				return err
			}
			*f = FlexInt(int64(fv))
			return nil
		}
		*f = FlexInt(v)
		return nil
	}

	var v int64
	if err := json.Unmarshal(b, &v); err == nil {
		*f = FlexInt(v)
		return nil
	}

	var fv float64
	if err := json.Unmarshal(b, &fv); err != nil {
		return err
	}
	*f = FlexInt(int64(fv))
	return nil
}

func (f FlexInt) Int64() int64 { return int64(f) }
