package main

import (
	"encoding/binary"
	"math"
)

const (
	F32Bin = iota + 1
	U64Bin = iota + 1
)

func EncodeU64(values []uint64) []byte {
	// version + length + content
	bin := make([]byte, 1+4+len(values)*8)
	bin[0] = U64Bin
	binary.LittleEndian.PutUint32(bin[1:], uint32(len(values)))
	for i, value := range values {
		binary.LittleEndian.PutUint64(bin[1+4+8*i:], value)
	}
	return bin
}

func EncodeF32(values []float32) []byte {
	// version + length + content
	bin := make([]byte, 1+4+len(values)*4)
	bin[0] = F32Bin
	binary.LittleEndian.PutUint32(bin[1:], uint32(len(values)))
	for i, value := range values {
		binary.LittleEndian.PutUint32(bin[1+4+4*i:], math.Float32bits(value))
	}
	return bin
}
