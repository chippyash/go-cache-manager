package valkey_test

import (
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/chippyash/go-cache-manager/adapter"
	"github.com/chippyash/go-cache-manager/adapter/valkey"
	"github.com/chippyash/go-cache-manager/errors"
	"github.com/chippyash/go-cache-manager/storage"
	"github.com/stretchr/testify/assert"
	valkey2 "github.com/valkey-io/valkey-go"
	"maps"
	"slices"
	"strconv"
	"testing"
	"time"
)

/** NB - We are using miniredis to mock the redis server. This means we cannot use client side caching **/

func TestValkeyAdapter_GetAndSetItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("key", "value")
	assert.True(t, ok)
	assert.NoError(t, err)

	val, err := sut.GetItem("key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestValkeyAdapter_GetAndSetItemManaged(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, true)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("key", "value")
	assert.True(t, ok)
	assert.NoError(t, err)

	//check that we have a data management entry in cache
	typ, err := rs.Get("gcm:key")
	assert.NoError(t, err)
	i64, err := strconv.ParseInt(typ, 10, 64)
	assert.NoError(t, err)
	assert.Equal(t, int64(storage.TypeString), i64)

	val, err := sut.GetItem("key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
	assert.Equal(t, fmt.Sprintf("%T", val), "string")
}

func TestValkeyAdapter_GetUnknownItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	val, err := sut.GetItem("key")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
	assert.Equal(t, nil, val)
}

func TestValkeyAdapter_GetAndSetMultipleItems(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	vals := map[string]any{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	keys, err := sut.SetItems(vals)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"key1", "key2", "key3"}, keys)

	ret, err := sut.GetItems(keys)
	assert.NoError(t, err)
	//NB - Redis only stores strings
	retVals := map[string]any{
		"key1": "value1",
		"key2": "2",
		"key3": "true",
	}
	assert.Equal(t, retVals, ret)
}

func TestValkeyAdapter_GetAndSetMultipleItemsManaged(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, true)
	sut, err := sut.Open()
	assert.NoError(t, err)
	tm, err := time.Parse(time.RFC3339, "2025-01-14T13:07:00+01:00")
	assert.NoError(t, err)
	vals := map[string]any{
		"TypeBoolean":   true,
		"TypeInteger":   2,
		"TypeInteger8":  int8(8),
		"TypeInteger16": int16(16),
		"TypeInteger32": int32(32),
		"TypeInteger64": int64(64),
		"TypeUint":      uint(2),
		"TypeUint8":     uint8(8),
		"TypeUint16":    uint16(16),
		"TypeUint32":    uint32(32),
		"TypeUint64":    uint64(64),
		"TypeFloat32":   float32(32.6),
		"TypeFloat64":   float64(64.6),
		"TypeString":    "value",
		"TypeDuration":  time.Second,
		"TypeTime":      tm,
		"TypeBytes":     []byte("value"),
	}
	keys, err := sut.SetItems(vals)
	assert.NoError(t, err)
	expectedKeys := slices.Collect(maps.Keys(vals))
	assert.ElementsMatch(t, expectedKeys, keys)

	//check that we have a data management entries in cache
	typs := map[string]int{
		"gcm:TypeBoolean":   storage.TypeBoolean,
		"gcm:TypeInteger":   storage.TypeInteger,
		"gcm:TypeInteger8":  storage.TypeInteger8,
		"gcm:TypeInteger16": storage.TypeInteger16,
		"gcm:TypeInteger32": storage.TypeInteger32,
		"gcm:TypeInteger64": storage.TypeInteger64,
		"gcm:TypeUint":      storage.TypeUint,
		"gcm:TypeUint8":     storage.TypeUint8,
		"gcm:TypeUint16":    storage.TypeUint16,
		"gcm:TypeUint32":    storage.TypeUint32,
		"gcm:TypeUint64":    storage.TypeUint64,
		"gcm:TypeFloat32":   storage.TypeFloat32,
		"gcm:TypeFloat64":   storage.TypeFloat64,
		"gcm:TypeString":    storage.TypeString,
		"gcm:TypeDuration":  storage.TypeDuration,
		"gcm:TypeTime":      storage.TypeTime,
		"gcm:TypeBytes":     storage.TypeBytes,
	}
	for k, tp := range typs {
		typ, err2 := rs.Get(k)
		assert.NoError(t, err2)
		i64, err2 := strconv.ParseInt(typ, 10, 64)
		assert.NoError(t, err2)
		assert.Equal(t, int64(tp), i64)
	}

	ret, err := sut.GetItems(keys)
	assert.NoError(t, err)
	assert.Equal(t, vals, ret)

	//check that the []byte value was serialized properly
	byt, err := rs.Get("TypeBytes")
	assert.NoError(t, err)
	assert.Equal(t, "\b\n\u0000\u0005value", byt)
}

func TestValkeyAdapter_HasItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)

	assert.True(t, sut.HasItem("foo"))
	assert.False(t, sut.HasItem("bop"))
}

func TestValkeyAdapter_HasMultipleItems(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	vals := map[string]any{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	_, err = sut.SetItems(vals)
	assert.NoError(t, err)

	ret := sut.HasItems([]string{"key1", "key2", "key3", "key4"})
	assert.True(t, ret["key1"])
	assert.True(t, ret["key2"])
	assert.True(t, ret["key3"])
	assert.False(t, ret["key4"])

}

func TestValkeyAdapter_Chaining(t *testing.T) {
	rs := miniRedis(t)
	rs2 := miniredis.RunT(t)
	chainedAdapter := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	chainedAdapter, err := chainedAdapter.Open()
	assert.NoError(t, err)
	vals := map[string]any{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	_, err = chainedAdapter.SetItems(vals)
	assert.NoError(t, err)
	//check that chained server has the keys
	chainKeys := rs.Keys()
	assert.True(t, slices.Contains(chainKeys, "one:key1"))
	assert.True(t, slices.Contains(chainKeys, "one:key2"))
	assert.True(t, slices.Contains(chainKeys, "one:key3"))

	sut := valkey.New("two:", rs2.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err = sut.Open()
	assert.NoError(t, err)
	sut.(storage.Chainable).ChainAdapter(chainedAdapter)
	//check that the parent server does not have keys
	parentKeys := rs2.Keys()
	assert.False(t, slices.Contains(parentKeys, "two:key1"))
	assert.False(t, slices.Contains(parentKeys, "two:key2"))
	assert.False(t, slices.Contains(parentKeys, "two:key3"))

	_, err = sut.GetItems([]string{"key1", "key2", "key3"})
	assert.NoError(t, err)
	//check that the parent adapter now has keys
	parentKeys = rs2.Keys()
	assert.True(t, slices.Contains(parentKeys, "two:key1"))
	assert.True(t, slices.Contains(parentKeys, "two:key2"))
	assert.True(t, slices.Contains(parentKeys, "two:key3"))
}

func TestValkeyAdapter_CheckAndSetItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)

	ok, err = sut.CheckAndSetItem("foo", "baz")
	assert.True(t, ok)
	assert.NoError(t, err)
	val, err := sut.GetItem("foo")
	assert.NoError(t, err)
	assert.Equal(t, "baz", val)

	ok, err = sut.CheckAndSetItem("bop", "baz")
	assert.False(t, ok)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), errors.ErrKeyNotFound.Error())
}

func TestValkeyAdapter_CheckAndSetMultipleItems(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)
	ok, err = sut.SetItem("bar", "bop")
	assert.True(t, ok)
	assert.NoError(t, err)

	ret, err := sut.CheckAndSetItems(map[string]any{"foo": "va1", "bar": 2, "bop": "not exists"})
	assert.Error(t, err)
	assert.True(t, slices.Contains(ret, "foo"))
	assert.True(t, slices.Contains(ret, "bar"))
	assert.False(t, slices.Contains(ret, "bop"))
}

func TestValkeyAdapter_TouchItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)

	assert.True(t, sut.TouchItem("foo"))
	assert.False(t, sut.TouchItem("bar"))
}

func TestValkeyAdapter_TouchMultipleItems(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	_, err = sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	keys := sut.TouchItems([]string{"foo", "bar", "bop"})
	assert.True(t, slices.Contains(keys, "foo"))
	assert.True(t, slices.Contains(keys, "bar"))
	assert.False(t, slices.Contains(keys, "bop"))
}

func TestValkeyAdapter_RemoveItem(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	_, err = sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	assert.True(t, sut.RemoveItem("foo"))
	val, err := sut.GetItem("foo")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
	assert.Equal(t, nil, val)

	val, err = sut.GetItem("bar")
	assert.NoError(t, err)
	assert.Equal(t, "bop", val)
}

func TestValkeyAdapter_RemoveMultipleItems(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	_, err = sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	keys := sut.RemoveItems([]string{"foo", "bar"})
	assert.True(t, slices.Contains(keys, "foo"))
	assert.True(t, slices.Contains(keys, "bar"))
}

func TestValkeyAdapter_IncrementValidNumber(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	keys := map[string]any{
		"int":     100,
		"int8":    int8(100),
		"int16":   int16(100),
		"int32":   int32(100),
		"int64":   int64(100),
		"uint":    uint(100),
		"uint8":   uint8(100),
		"uint16":  uint16(100),
		"uint32":  uint32(100),
		"uint64":  uint64(100),
		"uintptr": uintptr(100),
		"float32": float32(100),
		"float64": float64(100),
	}
	_, err = sut.SetItems(keys)
	assert.NoError(t, err)
	for k := range keys {
		val, err := sut.Increment(k, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(101), val)
	}
}

func TestValkeyAdapter_IncrementInvalidNumber(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	_, err = sut.SetItem("foo", "bar")
	assert.NoError(t, err)
	val, err := sut.Increment("foo", 1)
	assert.Error(t, err)
	assert.Equal(t, "value is not an integer or out of range", err.Error())
	assert.Equal(t, int64(0), val)
}

func TestValkeyAdapter_DecrementValidNumber(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	keys := map[string]any{
		"int":     100,
		"int8":    int8(100),
		"int16":   int16(100),
		"int32":   int32(100),
		"int64":   int64(100),
		"uint":    uint(100),
		"uint8":   uint8(100),
		"uint16":  uint16(100),
		"uint32":  uint32(100),
		"uint64":  uint64(100),
		"uintptr": uintptr(100),
		"float32": float32(100),
		"float64": float64(100),
	}
	_, err = sut.SetItems(keys)
	assert.NoError(t, err)
	for k := range keys {
		val, err := sut.Decrement(k, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(99), val)
	}
}

func TestValkeyAdapter_DecrementInvalidNumber(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	_, err = sut.SetItem("foo", "bar")
	assert.NoError(t, err)
	val, err := sut.Decrement("foo", 1)
	assert.Error(t, err)
	assert.Equal(t, "value is not an integer or out of range", err.Error())
	assert.Equal(t, int64(0), val)
}

func TestValkeyAdapter_GetClient(t *testing.T) {
	rs := miniRedis(t)
	sut := valkey.New("one:", rs.Addr(), time.Second*60, false, time.Second*0, false)
	sut, err := sut.Open()
	assert.NoError(t, err)
	client := sut.(*adapter.AbstractAdapter).Client.(valkey2.Client)
	assert.NotNil(t, client)
}

func miniRedis(t *testing.T) *miniredis.Miniredis {
	s := miniredis.NewMiniRedis()
	err := s.StartAddr(":6370")
	assert.NoError(t, err)
	t.Cleanup(s.Close)
	return s
}
