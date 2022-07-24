package db

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"strings"
)

type Converter[Value any] interface {
	convertTo(val Value) []byte
	convertFrom(val []byte) Value
}

type U32Converter struct{}

func (u U32Converter) convertTo(val uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], val)
	return buf[:]
}

func (u U32Converter) convertFrom(val []byte) uint32 {
	return binary.LittleEndian.Uint32(val)
}

type StringConverter struct{}

func (u StringConverter) convertTo(val string) []byte {
	return []byte(val)
}

func (u StringConverter) convertFrom(val []byte) string {
	return string(val)
}

type LowercaseStringConverter struct {
	StringConverter
}

func (u LowercaseStringConverter) convertTo(val string) []byte {
	return []byte(strings.ToLower(val))
}

type StructConverter[S any] struct{}

func (s StructConverter[S]) convertTo(val S) []byte {
	result := bytes.Buffer{}
	gob.NewEncoder(&result).Encode(val)
	return result.Bytes()
}

func (s StructConverter[S]) convertFrom(val []byte) (result S) {
	gob.NewDecoder(bytes.NewReader(val)).Decode(&result)
	return
}
