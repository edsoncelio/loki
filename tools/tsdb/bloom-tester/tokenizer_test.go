package main

import (
	"encoding/binary"
	"github.com/grafana/loki/pkg/logproto"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNGramTokenizer(t *testing.T) {
	tokenizer := threeSkip2
	for _, tc := range []struct {
		desc  string
		input string
		exp   []TokenB
	}{
		{
			desc:  "empty",
			input: "",
			exp:   []TokenB{},
		},
		{
			desc:  "single char",
			input: "a",
			exp:   []TokenB{},
		},
		{
			desc:  "four chars",
			input: "abcd",
			exp:   []TokenB{{Key: []byte("abc"), Value: "abc"}},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.exp, tokenizer.Tokens(tc.input))
		})
	}
}

func makeBuf(from, through, checksum int) []byte {
	p := make([]byte, 0, 256)
	i64buf := make([]byte, binary.MaxVarintLen64)
	i32buf := make([]byte, 4)

	binary.PutVarint(i64buf, int64(from))
	p = append(p, i64buf...)
	p = append(p, 58)
	binary.PutVarint(i64buf, int64(through))
	p = append(p, i64buf...)
	p = append(p, 58)
	binary.LittleEndian.PutUint32(i32buf, uint32(checksum))
	p = append(p, i32buf...)
	p = append(p, 58)
	return p
}

func TestWrappedTokenizer(t *testing.T) {
	tokenizer := threeSkip2
	for _, tc := range []struct {
		desc  string
		input string
		exp   []TokenB
	}{
		{
			desc:  "empty",
			input: "",
			exp:   []TokenB{},
		},
		{
			desc:  "single char",
			input: "a",
			exp:   []TokenB{},
		},
		{
			desc:  "four chars",
			input: "abcd",
			exp: []TokenB{
				{Key: append(makeBuf(0, 999999, 1), []byte("abc")...), Value: string(makeBuf(0, 999999, 1)) + "abc"},
				{Key: []byte("abc"), Value: "abc"}},
		},
		{
			desc:  "uuid",
			input: "2b1a5e46-36a2-4694-a4b1-f34cc7bdfc45",
			exp: []TokenB{
				{Key: append(makeBuf(0, 999999, 1), []byte("2b1")...), Value: string(makeBuf(0, 999999, 1)) + "2b1"},
				{Key: append(makeBuf(0, 999999, 1), []byte("a5e")...), Value: string(makeBuf(0, 999999, 1)) + "a5e"},
				{Key: append(makeBuf(0, 999999, 1), []byte("46-")...), Value: string(makeBuf(0, 999999, 1)) + "46-"},
				{Key: append(makeBuf(0, 999999, 1), []byte("36a")...), Value: string(makeBuf(0, 999999, 1)) + "36a"},
				{Key: append(makeBuf(0, 999999, 1), []byte("2-4")...), Value: string(makeBuf(0, 999999, 1)) + "2-4"},
				{Key: append(makeBuf(0, 999999, 1), []byte("694")...), Value: string(makeBuf(0, 999999, 1)) + "694"},
				{Key: append(makeBuf(0, 999999, 1), []byte("-a4")...), Value: string(makeBuf(0, 999999, 1)) + "-a4"},
				{Key: append(makeBuf(0, 999999, 1), []byte("b1-")...), Value: string(makeBuf(0, 999999, 1)) + "b1-"},
				{Key: append(makeBuf(0, 999999, 1), []byte("f34")...), Value: string(makeBuf(0, 999999, 1)) + "f34"},
				{Key: append(makeBuf(0, 999999, 1), []byte("cc7")...), Value: string(makeBuf(0, 999999, 1)) + "cc7"},
				{Key: append(makeBuf(0, 999999, 1), []byte("bdf")...), Value: string(makeBuf(0, 999999, 1)) + "bdf"},
				{Key: append(makeBuf(0, 999999, 1), []byte("c45")...), Value: string(makeBuf(0, 999999, 1)) + "c45"},
				{Key: []byte("2b1"), Value: "2b1"},
				{Key: []byte("a5e"), Value: "a5e"},
				{Key: []byte("46-"), Value: "46-"},
				{Key: []byte("36a"), Value: "36a"},
				{Key: []byte("2-4"), Value: "2-4"},
				{Key: []byte("694"), Value: "694"},
				{Key: []byte("-a4"), Value: "-a4"},
				{Key: []byte("b1-"), Value: "b1-"},
				{Key: []byte("f34"), Value: "f34"},
				{Key: []byte("cc7"), Value: "cc7"},
				{Key: []byte("bdf"), Value: "bdf"},
				{Key: []byte("c45"), Value: "c45"},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			chunkTokenizer := ChunkIDTokenizer(logproto.ChunkRef{Fingerprint: 1,
				From:     0,
				Through:  999999,
				Checksum: 1,
			}, tokenizer)
			require.Equal(t, tc.exp, chunkTokenizer.Tokens(tc.input))
		})
	}
}
