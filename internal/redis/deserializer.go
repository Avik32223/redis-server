package redis

import (
	"fmt"
	"strconv"
	"strings"
)

func eatBulkString(data []byte, c int) (bytesAte int) {
	t := string(data[c:])
	// minimum valid empty string
	if len(t) < len("$0\r\n\r\n") {
		return
	}

	if rune(t[0]) != '$' {
		return
	}

	t = t[1:]
	tSplit := strings.SplitN(t, "\r\n", 3)
	if len(tSplit) < 3 {
		return
	}

	tLen, err := strconv.Atoi(tSplit[0])
	if err != nil {
		return
	}

	if tLen != len(tSplit[1]) {
		return
	}

	bytesAte = len(fmt.Sprintf("$%s\r\n%s\r\n", tSplit[0], tSplit[1]))
	return
}

func eatArray(data []byte, c int) (bytesAte int) {
	t := string(data[c:])
	// minimum valid empty string
	if len(t) < len("*0\r\n") {
		return
	}

	if rune(t[0]) != '*' {
		return
	}

	t = t[1:]
	tSplit := strings.SplitN(t, "\r\n", 2)
	if len(tSplit) < 2 {
		return
	}

	tLen, err := strconv.Atoi(tSplit[0])
	if err != nil {
		return
	}
	ct := 0
	totalBulkStrings := 0
	for totalBulkStrings < tLen {
		cc := eatBulkString([]byte(tSplit[1][ct:]), 0)
		if cc != 0 {
			ct += cc
			totalBulkStrings++
			continue
		}
		break
	}
	ct += len(fmt.Sprintf("*%s\r\n", tSplit[0]))
	if totalBulkStrings != tLen {
		return
	}
	bytesAte = c + ct
	return
}

func parseBulkString(s string) (res string, err error) {
	s = s[1:]
	return strings.SplitN(s, "\r\n", 3)[1], nil
}

func parseArray(s string) ([]any, error) {
	s = s[1:]
	elems := strings.SplitN(s, "\r\n", 2)[1]

	c := 0
	res := make([]any, 0)
	for c < len(elems) {
		r := []byte(elems[c:])
		if cc := eatBulkString(r, 0); cc != 0 {
			v, _ := parseBulkString(elems[c : c+cc])
			res = append(res, v)
			c += cc
			continue
		}

		return nil, fmt.Errorf("unknown element type in array: %#v", elems)
	}

	return res, nil
}
