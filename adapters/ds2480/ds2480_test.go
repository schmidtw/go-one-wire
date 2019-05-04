package ds2480

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchToFromBytes(t *testing.T) {

	type TestVector struct {
		Tree uint64
		Last int
		Data []byte
	}

	tests := []TestVector{
		{Tree: 0x01, Last: 0, Data: []byte{0x3}},
		{Tree: 0x01, Last: 1, Data: []byte{0x6}},
		{Tree: 0x01, Last: 64, Data: []byte{0x2}},
		{Tree: 0x01, Last: 63, Data: []byte{0x2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x40}},
	}

	assert := assert.New(t)

	for _, test := range tests {
		expect := make([]byte, 16)
		for i, _ := range test.Data {
			expect[i] = test.Data[i]
		}
		got := searchToBytes(test.Tree, test.Last)
		assert.Equal(expect, got)

		out, last := searchFromBytes(expect)
		assert.Equal(test.Tree, out)
		assert.Equal(test.Last, last)
	}
}
