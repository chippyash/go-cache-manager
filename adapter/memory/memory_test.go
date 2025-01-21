package memory_test

import (
	"github.com/chippyash/go-cache-manager/adapter"
	"github.com/chippyash/go-cache-manager/adapter/memory"
	"github.com/chippyash/go-cache-manager/errors"
	"github.com/chippyash/go-cache-manager/storage"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"maps"
	"slices"
	"testing"
	"time"
)

func TestMemoryAdapter_GetAndSetItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	ok, err := sut.SetItem("key", "value")
	assert.True(t, ok)
	assert.NoError(t, err)

	val, err := sut.GetItem("key")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestMemoryAdapter_GetUnknownItem(t *testing.T) {
	sut := memory.New("namespace:", time.Second*60, time.Second*120)

	val, err := sut.GetItem("key")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
	assert.Equal(t, nil, val)
}

func TestMemoryAdapter_GetAndSetMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
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

	ret, err := sut.GetItems(keys)
	assert.NoError(t, err)
	assert.Equal(t, vals, ret)
}

func TestMemoryAdapter_HasItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)

	assert.True(t, sut.HasItem("foo"))
	assert.False(t, sut.HasItem("bop"))
}

func TestMemoryAdapter_HasMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	vals := map[string]any{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	_, err := sut.SetItems(vals)
	assert.NoError(t, err)

	ret := sut.HasItems([]string{"key1", "key2", "key3", "key4"})
	assert.True(t, ret["key1"])
	assert.True(t, ret["key2"])
	assert.True(t, ret["key3"])
	assert.False(t, ret["key4"])

}

func TestMemoryAdapter_Chaining(t *testing.T) {
	chainedAdapter := memory.New("one:", time.Second*60, time.Second*120)
	vals := map[string]any{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	_, err := chainedAdapter.SetItems(vals)
	assert.NoError(t, err)

	sut := memory.New("two:", time.Second*60, time.Second*120)
	sut.(storage.Chainable).ChainAdapter(chainedAdapter)
	//check that the parent adapter does not have keys
	adapterClient := sut.(*adapter.AbstractAdapter).Client.(*cache.Cache)
	_, found := adapterClient.Get("two:key1")
	assert.False(t, found)
	_, found = adapterClient.Get("two:key2")
	assert.False(t, found)
	_, found = adapterClient.Get("two:key2")
	assert.False(t, found)

	ret, err := sut.GetItems([]string{"key1", "key2", "key3"})
	assert.NoError(t, err)
	assert.Equal(t, vals, ret)
	//check that the parent adapter now has keys
	_, found = adapterClient.Get("two:key1")
	assert.True(t, found)
	_, found = adapterClient.Get("two:key2")
	assert.True(t, found)
	_, found = adapterClient.Get("two:key2")
	assert.True(t, found)
}

func TestMemoryAdapter_CheckAndSetItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
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
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
}

func TestMemoryAdapter_CheckAndSetMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
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

func TestMemoryAdapter_TouchItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	ok, err := sut.SetItem("foo", "bar")
	assert.True(t, ok)
	assert.NoError(t, err)

	assert.True(t, sut.TouchItem("foo"))
	assert.False(t, sut.TouchItem("bar"))
}

func TestMemoryAdapter_TouchMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	keys := sut.TouchItems([]string{"foo", "bar", "bop"})
	assert.True(t, slices.Contains(keys, "foo"))
	assert.True(t, slices.Contains(keys, "bar"))
	assert.False(t, slices.Contains(keys, "bop"))
}

func TestMemoryAdapter_RemoveItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItems(map[string]any{
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

func TestMemoryAdapter_RemoveMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	keys := sut.RemoveItems([]string{"foo", "bar"})
	assert.True(t, slices.Contains(keys, "foo"))
	assert.True(t, slices.Contains(keys, "bar"))
}

func TestMemoryAdapter_IncrementValidNumber(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
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
	_, err := sut.SetItems(keys)
	assert.NoError(t, err)
	for k := range keys {
		val, err := sut.Increment(k, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(101), val)
	}
}

func TestMemoryAdapter_IncrementInvalidNumber(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItem("foo", "bar")
	assert.NoError(t, err)
	val, err := sut.Increment("foo", 1)
	assert.Error(t, err)
	assert.Equal(t, "The value for foo is not an integer", err.Error())
	assert.Equal(t, int64(0), val)
}

func TestMemoryAdapter_DecrementValidNumber(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
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
	_, err := sut.SetItems(keys)
	assert.NoError(t, err)
	for k := range keys {
		val, err := sut.Decrement(k, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(99), val)
	}
}

func TestMemoryAdapter_DecrementInvalidNumber(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItem("foo", "bar")
	assert.NoError(t, err)
	val, err := sut.Decrement("foo", 1)
	assert.Error(t, err)
	assert.Equal(t, "The value for foo is not an integer", err.Error())
	assert.Equal(t, int64(0), val)
}

func TestMemoryAdapter_GetClient(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	client := sut.(*adapter.AbstractAdapter).Client.(*cache.Cache)
	assert.IsType(t, cache.Cache{}, client)
}
