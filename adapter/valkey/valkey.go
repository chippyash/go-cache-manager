package valkey

import (
	"context"
	"fmt"
	errs "github.com/pkg/errors"
	"github.com/valkey-io/valkey-go"
	adapter2 "go-cache-manager/adapter"
	"go-cache-manager/errors"
	"go-cache-manager/storage"
	"maps"
	"slices"
	"strconv"
	"time"
)

const (
	//OptHost Redis/Valkey host name
	OptHost = iota + storage.OptDataTypes + 1
	//OptPort the port number for the server. Will default to 6379 if not supplied
	OptPort
	//OptClientCaching set true to use client side caching else false
	OptClientCaching //see https://redis.io/docs/latest/develop/reference/client-side-caching/
	//OptClientCachingTtl if OptClientCaching is true, then how long to keep the client side caching item
	OptClientCachingTtl
	//OptValkeyOptions Valkey client options
	OptValkeyOptions
)

func New(namespace string, host string, ttl time.Duration, clientCaching bool, clientCachingTtl time.Duration) storage.Storage {
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
		OptHost:                   host,
		OptPort:                   6379,
		OptClientCaching:          clientCaching,
		OptClientCachingTtl:       clientCachingTtl,
		OptValkeyOptions: valkey.ClientOption{
			InitAddress:  []string{host},
			DisableCache: !clientCaching,
		},
	}

	adapter := new(adapter2.AbstractAdapter)
	adapter.Name = "valkey"
	adapter.SetOptions(opts)

	anyToString := func(v any) string {
		return fmt.Sprintf("%v", v)
	}

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
			cl := adapter.Client.(valkey.Client)
			var val any
			var found bool
			switch adapter.GetOptions()[OptClientCaching].(bool) {
			case true:
				resp := cl.DoCache(
					context.TODO(),
					cl.B().Get().Key(nsKey).Cache(),
					adapter.GetOptions()[OptClientCachingTtl].(time.Duration),
				)
				found = resp.Error() == nil
				val, _ = resp.ToAny()
			case false:
				resp := cl.Do(
					context.TODO(),
					cl.B().Get().Key(nsKey).Build(),
				)
				found = resp.Error() == nil
				val, _ = resp.ToAny()
			}
			if !found {
				if adapter.GetChained() != nil {
					v, err2 := adapter.GetChained().GetItem(key)
					if err2 != nil {
						return nil, errors.ErrKeyNotFound
					}
					adapter.SetItem(key, v)
					return val, nil
				}
				return nil, errors.ErrKeyNotFound
			}
			return val, nil
		}).
		SetGetItemsFunc(func(keys []string) (map[string]any, error) {
			ret := make(map[string]any)
			cl := adapter.Client.(valkey.Client)
			var err2 error
			switch adapter.GetOptions()[OptClientCaching].(bool) {
			case true:
				cmds := make([]valkey.CacheableTTL, 0, len(keys))
				for _, key := range keys {
					nsKey := adapter.NamespacedKey(key)
					if !adapter.ValidateKey(nsKey) {
						return ret, errs.Wrap(errors.ErrKeyInvalid, "failed to get item")
					}
					cmds = append(cmds, valkey.CT(cl.B().Get().Key(nsKey).Cache(), adapter.GetOptions()[OptClientCachingTtl].(time.Duration)))
				}
				for i, resp := range cl.DoMultiCache(context.TODO(), cmds...) {
					if resp.Error() == nil {
						v, err3 := resp.ToAny()
						ret[keys[i]] = v
						if err3 != nil {
							err2 = errs.Wrap(err3, "failed to get item")
						}
					}
				}
			case false:
				cmds := make(valkey.Commands, 0, len(keys))
				for _, key := range keys {
					nsKey := adapter.NamespacedKey(key)
					if !adapter.ValidateKey(nsKey) {
						return ret, errs.Wrap(errors.ErrKeyInvalid, "failed to get item")
					}
					cmds = append(cmds, cl.B().Get().Key(nsKey).Build())
				}
				for i, resp := range cl.DoMulti(context.TODO(), cmds...) {
					if resp.Error() != nil {
						if adapter.GetChained() != nil {
							v, err3 := adapter.GetChained().GetItem(keys[i])
							if err3 != nil {
								return ret, errs.Wrap(err3, "failed to get item")
							}
							adapter.SetItem(keys[i], v)
							ret[keys[i]] = v
							continue
						}
						err2 = errs.Wrap(resp.Error(), "failed to get item")
						continue
					}
					v, err3 := resp.ToAny()
					ret[keys[i]] = v
					if err3 != nil {
						err2 = errs.Wrap(err3, "failed to get item")
					}
				}
			}

			return ret, err2
		}).
		SetSetItemFunc(func(key string, value any) (bool, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false, errors.ErrKeyInvalid
			}
			cl := adapter.Client.(valkey.Client)
			err2 := cl.Do(
				context.TODO(),
				cl.B().Set().Key(nsKey).Value(anyToString(value)).Ex(adapter.GetOptions()[storage.OptTTL].(time.Duration)).Build(),
			).Error()
			if err2 != nil {
				return false, errs.Wrap(err2, "failed to set item")
			}
			if adapter.GetChained() != nil {
				_, _ = adapter.GetChained().SetItem(key, value)
			}
			return true, nil
		}).
		SetSetItemsFunc(func(values map[string]any) ([]string, error) {
			keys := make([]string, len(values))
			cmds := make(valkey.Commands, 0, len(keys))
			cl := adapter.Client.(valkey.Client)
			var err error
			for key, value := range values {
				nsKey := adapter.NamespacedKey(key)
				if !adapter.ValidateKey(nsKey) {
					return keys, errors.ErrKeyInvalid
				}
				cmds = append(cmds, cl.B().Set().Key(nsKey).Value(anyToString(value)).Ex(adapter.GetOptions()[storage.OptTTL].(time.Duration)).Build())
			}

			responses := cl.DoMulti(context.TODO(), cmds...)
			keyNames := slices.Collect(maps.Keys(values))
			for i := 0; i < len(values); i++ {
				if responses[i].Error() != nil {
					err = errs.Wrap(responses[i].Error(), "failed to set item")
					continue
				}

				keys[i] = keyNames[i]
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
			cl := adapter.Client.(valkey.Client)
			resp := cl.Do(context.TODO(), cl.B().Exists().Key(nsKey).Build())
			if resp.Error() != nil {
				return false
			}
			v, err := resp.AsInt64()
			if err != nil {
				return false
			}
			if v == 0 && adapter.GetChained() != nil {
				return adapter.GetChained().HasItem(key)
			}
			return v == 1
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
			cl := adapter.Client.(valkey.Client)
			err2 := cl.Do(
				context.TODO(),
				cl.B().Set().Key(nsKey).Value(anyToString(value)).Xx().Ex(adapter.GetOptions()[storage.OptTTL].(time.Duration)).Build(),
			).Error()
			if err2 != nil {
				return false, errs.Wrap(err2, errors.ErrKeyNotFound.Error())
			}
			if adapter.GetChained() != nil {
				return adapter.GetChained().CheckAndSetItem(key, value)
			}
			return err2 == nil, err2
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
			cl := adapter.Client.(valkey.Client)
			err2 := cl.Do(
				context.TODO(),
				cl.B().Del().Key(nsKey).Build(),
			).Error()
			if adapter.GetChained() != nil {
				return adapter.GetChained().RemoveItem(key)
			}
			return err2 == nil
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
			cl := adapter.Client.(valkey.Client)
			err := cl.Do(
				context.TODO(),
				cl.B().Incrby().Key(nsKey).Increment(n).Build(),
			).Error()
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
			return strconv.ParseInt(val.(string), 10, 64)
		}).
		SetDecrementFunc(func(key string, n int64) (int64, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return 0, errors.ErrNotWritable
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return 0, errors.ErrKeyInvalid
			}
			cl := adapter.Client.(valkey.Client)
			err := cl.Do(
				context.TODO(),
				cl.B().Decrby().Key(nsKey).Decrement(n).Build(),
			).Error()
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
			return strconv.ParseInt(val.(string), 10, 64)
		}).
		SetOpenFunc(func() (storage.Storage, error) {
			c, err := valkey.NewClient(
				adapter.GetOptions()[OptValkeyOptions].(valkey.ClientOption),
			)
			if err != nil {
				return nil, errs.Wrap(err, "failed to create Valkey client")
			}
			adapter.Client = c

			return adapter, err
		}).
		SetCloseFunc(func() error {
			return nil
		})

	return adapter
}
