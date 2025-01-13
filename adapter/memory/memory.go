package memory

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	errs "github.com/pkg/errors"
	adapter2 "go-cache-manager/adapter"
	"go-cache-manager/errors"
	"go-cache-manager/storage"
	"time"
)

const (
	//OptPurgeTtl expires any cache item older than a time.Duration
	OptPurgeTtl = iota + storage.OptDataTypes + 1
)

func New(namespace string, ttl, purgeTtl time.Duration) storage.Storage {
	//set the options
	dTypes := storage.DefaultDataTypes
	opts := storage.StorageOptions{
		storage.OptNamespace:      namespace,
		storage.OptKeyPattern:     "",
		storage.OptReadable:       true,
		storage.OptWritable:       true,
		storage.OptTTL:            ttl,
		storage.OptMaxKeyLength:   0,
		storage.OptMaxValueLength: 0,
		storage.OptDataTypes:      dTypes,
		OptPurgeTtl:               purgeTtl,
	}

	adapter := new(adapter2.AbstractAdapter)
	adapter.Name = "memory"
	adapter.Client = cache.New(ttl, purgeTtl)
	adapter.SetOptions(opts)

	//set the functions
	adapter.
		SetGetItemFunc(func(key string) (any, error) {
			if !adapter.GetOptions()[storage.OptReadable].(bool) {
				return nil, errors.ErrNotReadable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return nil, errors.ErrKeyInvalid
			}
			val, found := adapter.Client.(*cache.Cache).Get(nsKey)
			if !found {
				if adapter.GetChained() != nil {
					val, err := adapter.GetChained().GetItem(key)
					if err != nil {
						return nil, errors.ErrKeyNotFound
					}
					adapter.SetItem(key, val)
					return val, nil
				}
				return nil, errors.ErrKeyNotFound
			}
			return val, nil
		}).
		SetGetItemsFunc(func(keys []string) (map[string]any, error) {
			ret := make(map[string]any)
			var err error
			for _, key := range keys {
				val, found := adapter.GetItem(key)
				if found != nil {
					if errs.Is(found, errors.ErrKeyInvalid) || errs.Is(found, errors.ErrNotReadable) {
						err = found
						continue
					}
					err = errors.ErrKeyNotFound
					continue
				}
				ret[key] = val
			}
			return ret, err
		}).
		SetSetItemFunc(func(key string, value any) (bool, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false, errors.ErrKeyInvalid
			}
			adapter.Client.(*cache.Cache).Set(nsKey, value, adapter.GetOptions()[storage.OptTTL].(time.Duration))
			if adapter.GetChained() != nil {
				_, _ = adapter.GetChained().SetItem(key, value)
			}
			return true, nil
		}).
		SetSetItemsFunc(func(values map[string]any) ([]string, error) {
			keys := make([]string, len(values))
			i := 0
			var err error
			for key, value := range values {
				_, e := adapter.SetItem(key, value)
				if e != nil {
					err = e
				}
				keys[i] = key
				i++
			}
			return keys, err
		}).
		SetHasItemFunc(func(key string) bool {
			if !adapter.GetOptions()[storage.OptReadable].(bool) {
				return false
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false
			}
			_, found := adapter.Client.(*cache.Cache).Get(nsKey)
			if !found && adapter.GetChained() != nil {
				return adapter.GetChained().HasItem(key)
			}
			return found
		}).
		SetHasItemsFunc(func(keys []string) map[string]bool {
			ret := make(map[string]bool)
			for _, key := range keys {
				ret[key] = adapter.HasItem(key)
			}
			return ret
		}).
		SetCheckAndSetItemFunc(func(key string, value any) (bool, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false, errors.ErrKeyInvalid
			}
			err := adapter.Client.(*cache.Cache).Replace(nsKey, value, adapter.GetOptions()[storage.OptTTL].(time.Duration))
			if err != nil {
				err = errors.ErrKeyNotFound
			}
			if adapter.GetChained() != nil {
				return adapter.GetChained().CheckAndSetItem(key, value)
			}
			return err == nil, err
		}).
		SetCheckAndSetItemsFunc(func(values map[string]any) ([]string, error) {
			keys := make([]string, 0)
			var err error
			for key, value := range values {
				ok, _ := adapter.CheckAndSetItem(key, value)
				if !ok {
					err = errors.ErrKeyNotFound
					continue
				}
				keys = append(keys, key)
			}
			return keys, err
		}).
		SetTouchItemFunc(func(key string) bool {
			if !adapter.GetOptions()[storage.OptReadable].(bool) {
				return false
			}
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false
			}
			val, err := adapter.GetItem(key)
			if err != nil {
				return false
			}
			ok, err := adapter.SetItem(key, val)
			if err != nil {
				return false
			}
			if adapter.GetChained() != nil {
				return adapter.GetChained().TouchItem(key)
			}
			return ok
		}).
		SetTouchItemsFunc(func(keys []string) []string {
			ret := make([]string, 0)
			for _, key := range keys {
				if adapter.TouchItem(key) {
					ret = append(ret, key)
				}
			}
			return ret
		}).
		SetRemoveItemFunc(func(key string) bool {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false
			}
			adapter.Client.(*cache.Cache).Delete(nsKey)
			if adapter.GetChained() != nil {
				return adapter.GetChained().RemoveItem(key)
			}
			return true
		}).
		SetRemoveItemsFunc(func(keys []string) []string {
			for _, key := range keys {
				adapter.RemoveItem(key)
			}
			return keys
		}).
		SetIncrementFunc(func(key string, n int64) (int64, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return 0, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return 0, errors.ErrKeyInvalid
			}
			err := adapter.Client.(*cache.Cache).Increment(nsKey, n)
			if err != nil {
				return 0, err
			}
			if adapter.GetChained() != nil {
				return adapter.GetChained().Increment(key, n)
			}
			val, err := adapter.GetItem(key)
			if err != nil {
				return 0, err
			}
			switch val.(type) {
			case int:
				return int64(val.(int)), nil
			case int8:
				return int64(val.(int8)), nil
			case int16:
				return int64(val.(int16)), nil
			case int32:
				return int64(val.(int32)), nil
			case int64:
				return val.(int64), nil
			case uint:
				return int64(val.(uint)), nil
			case uintptr:
				return int64(val.(uintptr)), nil
			case uint8:
				return int64(val.(uint8)), nil
			case uint16:
				return int64(val.(uint16)), nil
			case uint32:
				return int64(val.(uint32)), nil
			case uint64:
				return int64(val.(uint64)), nil
			case float32:
				return int64(val.(float32)), nil
			case float64:
				return int64(val.(float64)), nil
			default:
				return 0, fmt.Errorf("value for %s is not an integer or integer like", key)
			}
		}).
		SetDecrementFunc(func(key string, n int64) (int64, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return 0, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return 0, errors.ErrKeyInvalid
			}
			err := adapter.Client.(*cache.Cache).Decrement(nsKey, n)
			if err != nil {
				return 0, err
			}
			if adapter.GetChained() != nil {
				return adapter.GetChained().Decrement(key, n)
			}
			val, err := adapter.GetItem(key)
			if err != nil {
				return 0, err
			}
			switch val.(type) {
			case int:
				return int64(val.(int)), nil
			case int8:
				return int64(val.(int8)), nil
			case int16:
				return int64(val.(int16)), nil
			case int32:
				return int64(val.(int32)), nil
			case int64:
				return val.(int64), nil
			case uint:
				return int64(val.(uint)), nil
			case uintptr:
				return int64(val.(uintptr)), nil
			case uint8:
				return int64(val.(uint8)), nil
			case uint16:
				return int64(val.(uint16)), nil
			case uint32:
				return int64(val.(uint32)), nil
			case uint64:
				return int64(val.(uint64)), nil
			case float32:
				return int64(val.(float32)), nil
			case float64:
				return int64(val.(float64)), nil
			default:
				return 0, fmt.Errorf("value for %s is not an integer or integer like", key)
			}
		}).
		SetOpenFunc(func() (storage.Storage, error) {
			return adapter, nil
		}).
		SetCloseFunc(func() error {
			return nil
		})

	return adapter
}
