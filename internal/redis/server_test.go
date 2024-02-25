package redis

import "testing"

func Test_eatBulkString(t *testing.T) {
	type testCase struct {
		name string
		data []byte
		c    int
		want int
	}
	tests := []testCase{
		// TODO: Add test cases.
		{"test-1", []byte("$0\r\n\r\n"), 0, 6},
		{"test-2", []byte("$5\r\nhello\r\n"), 0, 11},
		{"test-3", []byte("$5\r\nhello\r\nthere"), 0, 11},
		{"test-4", []byte("$0\r\nhel\r\nthere"), 0, 0},
		{"test-5", []byte("$5\r\nhelllllo\r\nthere"), 0, 0},
		{"test-6", []byte("$55\r\nhelllllo\r\nthere"), 0, 0},
		{"test-7", []byte("$\r\nhelllllo\r\nthere"), 0, 0},
		{"test-8", []byte("5\r\nhelllllo\r\nthere"), 0, 0},
		{"test-9", []byte("$a\r\nhello\r\nthere\r\n"), 0, 0},
		{"test-10", []byte("$5\r\nhellothere"), 0, 0},
		{"test-11", []byte("random$5\r\nhello\r\n"), 6, 11},
		{"test-12", []byte("random"), 0, 0},
		{"test-13", []byte("$6\r\nrandom"), 0, 0},
		{"test-14", []byte("$6\r\nrando\r\n"), 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eatBulkString(tt.data, tt.c); got != tt.want {
				t.Errorf("eatBulkString() = %v, want %v", got, tt.want)
			}
		})
	}
}
