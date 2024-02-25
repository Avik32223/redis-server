package redis

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"unicode"

	"github.com/Avik32223/redis-server/pkg/lists"
)

func SerializeSimpleString(m string, opts *SerDeOpts) (string, error) {
	return fmt.Sprintf("+%s\r\n", m), nil
}

func SerializeSimpleError(err error, opts *SerDeOpts) (string, error) {
	return fmt.Sprintf("-%s\r\n", err), nil
}

func SerializeInt(m int64, opts *SerDeOpts) (string, error) {
	return fmt.Sprintf(":%d\r\n", m), nil
}

func SerializeBulkString(m string, opts *SerDeOpts) (string, error) {
	return fmt.Sprintf("$%d\r\n%s\r\n", len(m), m), nil
}

func SerializeNull(opts *SerDeOpts) (string, error) {
	return "$-1\r\n", nil
}

func SerializeArray(m []any, opts *SerDeOpts) (string, error) {
	s := new(strings.Builder)
	s.WriteString(fmt.Sprintf("*%d\r\n", len(m)))
	for _, i := range m {
		result, err := Serialize(i, opts)
		if err != nil {
			return "", err
		}
		s.WriteString(result)
	}
	return s.String(), nil
}

func SerializeBoolean(m bool, opts *SerDeOpts) (string, error) {
	if m {
		return "#t\r\n", nil
	}
	return "#f\r\n", nil
}

func SerializeDouble(m float64, opts *SerDeOpts) (string, error) {
	if math.IsInf(m, 1) {
		return ",inf\r\n", nil
	}
	if math.IsInf(m, -1) {
		return ",-inf\r\n", nil
	}
	if math.IsNaN(m) {
		return ",nan\r\n", nil
	}
	return fmt.Sprintf(",%e\r\n", m), nil
}

func SerializeBigNumber(m any, opts *SerDeOpts) (string, error) {
	switch m := m.(type) {
	case big.Int:
		return fmt.Sprintf("(%s\r\n", m.String()), nil
	case big.Float:
		return fmt.Sprintf("(%s\r\n", m.String()), nil
	case big.Rat:
		return fmt.Sprintf("(%s\r\n", m.String()), nil
	}
	return "", fmt.Errorf("big number type not supported: %#v", m)
}

func SerializeBulkError(m error, opts *SerDeOpts) (string, error) {
	return fmt.Sprintf("!%d\r\n%s\r\n", len(m.Error()), m), nil
}

func SerializeVerbatimString(m string, opts *SerDeOpts) (string, error) {
	return "", nil
}

func SerializeMap(m map[string]any, opts *SerDeOpts) (string, error) {
	resp_version, ok := (*opts)["resp_version"]
	if ok && resp_version == "2" {
		l := make([]any, 0)
		for k, v := range m {
			l = append(l, k)
			l = append(l, v)
		}
		return SerializeArray(l, opts)
	}

	s := new(strings.Builder)
	s.WriteRune('%')
	s.WriteString(fmt.Sprintf("%d\r\n", len(m)))
	for k, v := range m {
		kS, err := Serialize(k, opts)
		if err != nil {
			return "", err
		}
		kV, err := Serialize(v, opts)
		if err != nil {
			return "", err
		}
		s.WriteString(kS)
		s.WriteString(kV)
	}
	return s.String(), nil
}

func SerializeError(m error, opts *SerDeOpts) (string, error) {
	bulkError := strings.ContainsFunc(m.Error(), func(r rune) bool {
		if unicode.IsControl(r) {
			return true
		}
		if r == '\n' || r == '\r' {
			return true
		}
		return false
	})
	if bulkError {
		return SerializeBulkError(m, opts)
	}
	return SerializeSimpleError(m, opts)

}

func SerializeString(m string, opts *SerDeOpts) (string, error) {
	verbatim, ok := (*opts)["verbatim"]
	if ok && len(verbatim) == 3 {
		return SerializeVerbatimString(m, opts)
	}
	isBulkString := strings.ContainsFunc(m, func(r rune) bool {
		if unicode.IsControl(r) {
			return true
		}
		if r == '\n' || r == '\r' {
			return true
		}
		return false
	})
	if isBulkString {
		return SerializeBulkString(m, opts)
	}
	return SerializeSimpleString(m, opts)
}

func Serialize(m any, opts *SerDeOpts) (string, error) {
	if opts == nil {
		opts = &defaultSerdeOpts
	}

	if m == nil {
		return SerializeNull(opts)
	}

	rv := reflect.ValueOf(m)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return SerializeInt(rv.Int(), opts)
	case reflect.Float32, reflect.Float64:
		return SerializeDouble(rv.Float(), opts)
	case reflect.Bool:
		return SerializeBoolean(rv.Bool(), opts)
	case reflect.Slice, reflect.Array:
		return SerializeArray(rv.Interface().([]any), opts)
	case reflect.Map:
		x := rv.Interface().(map[string]any)
		return SerializeMap(x, opts)
	case reflect.TypeOf(lists.List{}).Kind():
		v := rv.Interface().(lists.List)
		return SerializeArray(v.ToSlice(), opts)
	}

	switch mt := m.(type) {
	case big.Int, big.Float, big.Rat:
		return SerializeBigNumber(mt, opts)
	case string:
		return SerializeString(mt, opts)
	case error:
		return SerializeError(mt, opts)
	}
	return "", fmt.Errorf("failed to Serialize %#v", m)
}
