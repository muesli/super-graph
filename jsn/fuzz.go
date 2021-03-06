package jsn

import "bytes"

// FuzzerEntrypoint for Fuzzbuzz
func FuzzerEntryPoint(data []byte) int {
	err1 := Validate(string(data))

	var b1 bytes.Buffer
	err2 := Filter(&b1, data, []string{"id", "full_name", "embed"})

	path1 := [][]byte{[]byte("data"), []byte("users")}
	Strip(data, path1)

	from := []Field{
		{[]byte("__twitter_id"), []byte(`[{ "name": "hello" }, { "name": "world"}]`)},
		{[]byte("__twitter_id"), []byte(`"ABC123"`)},
	}

	to := []Field{
		{[]byte("__twitter_id"), []byte(`"1234567890"`)},
		{[]byte("some_list"), []byte(`[{"id":1,"embed":{"id":8}},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13}]`)},
	}

	var b2 bytes.Buffer
	err3 := Replace(&b2, data, from, to)

	Keys(data)

	if err1 != nil || err2 != nil || err3 != nil {
		return -1
	}

	return 0
}
