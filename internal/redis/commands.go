package redis

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Avik32223/redis-server/pkg/lists"
)

var listKind = reflect.TypeOf(lists.List{}).Kind()
var (
	errorKeyAbsent      = fmt.Errorf("key absent")
	errorInvalidCommand = fmt.Errorf("invalid command")
)

type Command func(State, ...any) (any, error)

var commandMap = map[string]Command{
	"get":     get,
	"set":     set,
	"command": command,
	"ping":    ping,
	"echo":    echo,
	"exists":  exists,
	"del":     del,
	"incr":    incr,
	"decr":    decr,
	"lpush":   lpush,
	"rpush":   rpush,
}

func invalidCommand(s State, ca ...any) (any, error) {
	return nil, errorInvalidCommand
}

func get(s State, ca ...any) (any, error) {
	if len(ca) != 1 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'get' command")
	}
	key := ca[0]
	data := *s.Data()
	switch k := key.(type) {
	case string:
		x, ok := data[k]
		if ok {
			if time.Now().Before(x.expiresAt) {
				return x.val, nil
			}
			delete(data, k)
		}
		return nil, errorKeyAbsent
	}
	return nil, fmt.Errorf("invalid use. key must be string")
}

func set(s State, ca ...any) (any, error) {
	if len(ca) < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'set' command")
	}
	now := time.Now()
	expiryFound := false
	expiresAt := now.Add(time.Duration(math.MaxInt64))
	for i := 2; i < len(ca); i++ {
		switch x := ca[i].(type) {
		case string:
			if !expiryFound && i+1 < len(ca) && slices.Index([]string{"EX", "PX", "EXAT", "PXAT"}, x) != -1 {
				amount, err := strconv.Atoi(ca[i+1].(string))
				if err != nil {
					return nil, fmt.Errorf("ERR wrong arguments for 'set' command. '%s' value invalid", x)
				}
				if x == "EX" {
					expiresAt = now.Add(time.Duration(amount) * time.Second)
				} else if x == "PX" {
					expiresAt = now.Add(time.Duration(amount) * time.Millisecond)
				} else if x == "EXAT" {
					expiresAt = time.Unix(int64(amount), 0)
				} else if x == "PXAT" {
					expiresAt = time.UnixMilli(int64(amount))
				}
				i++
			} else {
				return nil, fmt.Errorf("ERR wrong number of arguments for 'set' command. multiple expiries provided")
			}
		}
	}

	key := ca[0]
	value := ca[1]
	data := *s.Data()
	switch k := key.(type) {
	case string:
		data[k] = &stateValue{
			val:       value,
			expiresAt: expiresAt,
		}
	default:
		return nil, fmt.Errorf("invalid use. key must be string")
	}
	return "OK", nil
}

func exists(s State, ca ...any) (any, error) {
	if len(ca) < 1 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'exists' command")
	}
	c := 0
	for _, key := range ca {
		if _, err := get(s, key); err == nil {
			c++
		}
	}
	return c, nil
}

func del(s State, ca ...any) (any, error) {
	if len(ca) < 1 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'del' command")
	}
	c := 0
	data := *s.Data()
	for _, key := range ca {
		if _, err := get(s, key); err == nil {
			delete(data, key.(string))
			c++
		}
	}
	return c, nil
}

func incr(s State, ca ...any) (any, error) {
	if len(ca) < 1 || len(ca) > 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'incr' command")
	}
	v, err := get(s, ca[0])
	if err != nil {
		if err != errorKeyAbsent {
			return v, err
		}
		v = "0"
	}
	amount := 1
	if len(ca) == 2 {
		switch a := ca[1].(type) {
		case string:
			x, err := strconv.Atoi(a)
			if err != nil {
				return nil, fmt.Errorf("ERR invalid incr value provided")
			}
			amount = x
		}
	}
	switch vt := v.(type) {
	case string:
		i, err := strconv.Atoi(vt)
		if err != nil {
			return nil, err
		}
		nv := i + amount
		_, err = set(s, ca[0], fmt.Sprint(nv))
		if err != nil {
			return nil, err
		}
		return nv, nil
	}
	return nil, fmt.Errorf("ERR wrong number of arguments for 'incr' command")
}

func decr(s State, ca ...any) (any, error) {
	if len(ca) < 1 || len(ca) > 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'decr' command")
	}
	v, err := get(s, ca[0])
	if err != nil {
		if err != errorKeyAbsent {
			return v, err
		}
		v = "0"
	}
	amount := 1
	if len(ca) == 2 {
		switch a := ca[1].(type) {
		case string:
			x, err := strconv.Atoi(a)
			if err != nil {
				return nil, fmt.Errorf("ERR invalid 'decr' value provided")
			}
			amount = x
		}
	}
	switch vt := v.(type) {
	case string:
		i, err := strconv.Atoi(vt)
		if err != nil {
			return nil, err
		}
		nv := i - amount
		_, err = set(s, ca[0], fmt.Sprint(nv))
		if err != nil {
			return nil, err
		}
		return nv, nil
	}

	return nil, fmt.Errorf("ERR wrong number of arguments for 'decr' command")
}

func lpush(s State, ca ...any) (any, error) {
	if len(ca) < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'lpush' command")
	}
	key := ca[0]
	ca = ca[1:]
	keyAbsent := false
	val, err := get(s, key)
	if err != nil {
		if err != errorKeyAbsent {
			return val, err
		}
		keyAbsent = true
		val = *lists.NewList()
	}
	vt := reflect.ValueOf(val)
	switch vt.Kind() {
	case listKind:
		pv := vt.Interface().(lists.List)
		for i := 0; i < len(ca); i++ {
			pv.Prepend(ca[i])
		}
		if keyAbsent {
			if v, err := set(s, key, pv); err != nil {
				return v, err
			}
		} else {
			data := *s.Data()
			nVal := data[key.(string)]
			nVal.val = pv
		}
		return pv.Len(), nil
	}
	return nil, fmt.Errorf("ERR cannot push to a non list value")
}

func rpush(s State, ca ...any) (any, error) {
	if len(ca) < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'rpush' command")
	}
	key := ca[0]
	ca = ca[1:]
	keyAbsent := false
	val, err := get(s, key)
	if err != nil {
		if err != errorKeyAbsent {
			return val, err
		}
		keyAbsent = true
		val = *lists.NewList()
	}
	vt := reflect.ValueOf(val)
	switch vt.Kind() {
	case listKind:
		pv := vt.Interface().(lists.List)
		for i := 0; i < len(ca); i++ {
			pv.Append(ca[i])
		}
		if keyAbsent {
			if v, err := set(s, key, pv); err != nil {
				return v, err
			}
		} else {
			data := *s.Data()
			nVal := data[key.(string)]
			nVal.val = pv
		}
		return pv.Len(), nil
	}
	return nil, fmt.Errorf("ERR cannot push to a non list value")
}

func ping(s State, ca ...any) (any, error) {
	return "PONG", nil
}

func echo(s State, ca ...any) (any, error) {
	if len(ca) != 1 {
		return nil, fmt.Errorf("ERR wrong number of arguments for 'echo' command")
	}
	return ca[0], nil
}

func command(s State, ca ...any) (any, error) {
	return []any{}, nil
}

func newCommand(arr []any) Command {
	if len(arr) < 1 {
		return invalidCommand
	}

	switch cmdName := arr[0].(type) {
	case string:
		cmdName = strings.ToLower(cmdName)
		cmd, ok := commandMap[cmdName]
		if !ok {
			return invalidCommand
		}
		return cmd
	}

	return get
}

func RunCommand(s State, b []byte) (any, error) {
	sa := string(b)
	arr := make([]any, 0)
	if c := eatArray(b, 0); c != 0 {
		arr, _ = parseArray(sa)
	} else {
		for _, s := range strings.Split(sa, " ") {
			arr = append(arr, s)
		}
	}
	if len(arr) < 1 {
		return nil, errorInvalidCommand
	}
	cmd := newCommand(arr)
	return cmd(s, arr[1:]...)
}
