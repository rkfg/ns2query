package db

import (
	"bytes"

	"go.etcd.io/bbolt"
)

type Bucket[Key any, Value any] struct {
	*bbolt.Bucket
	keyConverter   Converter[Key]
	valueConverter Converter[Value]
}

func (b Bucket[Key, Value]) DeleteValue(key Key) error {
	return b.Delete(b.keyConverter.convertTo(key))
}

func (b Bucket[Key, Value]) PutValue(key Key, value Value) error {
	return b.Put(b.keyConverter.convertTo(key), b.valueConverter.convertTo(value))
}

func (b Bucket[Key, Value]) GetValue(key Key) (Value, error) {
	val := b.Get(b.keyConverter.convertTo(key))
	if val == nil {
		var zero Value
		return zero, ErrNotFound
	}
	return b.valueConverter.convertFrom(val), nil
}

func (b Bucket[Key, Value]) FindFirstValue(prefix Key) (result Value, err error) {
	c := b.Cursor()
	k, v := c.Seek(b.keyConverter.convertTo(prefix))
	if k == nil || !bytes.HasPrefix(k, b.keyConverter.convertTo(prefix)) {
		var zero Value
		return zero, ErrNotFound
	}
	return b.valueConverter.convertFrom(v), nil
}
