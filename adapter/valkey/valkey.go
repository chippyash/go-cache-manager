package valkey

import (
	"context"
	"fmt"
	adapter2 "github.com/chippyash/go-cache-manager/adapter"
	"github.com/chippyash/go-cache-manager/errors"
	"github.com/chippyash/go-cache-manager/storage"
	errs "github.com/pkg/errors"
	"github.com/valkey-io/valkey-go"
	"strconv"
	"strings"
	"time"
)

const (
	//OptHost Redis/Valkey host name. type: string
	OptHost = iota + storage.OptDataTypes + 1
	//OptPort the port number for the server. Will default to 6379 if not supplied. type: int
	OptPort
	//OptClientCaching set true to use client side caching else false. type: bool
	OptClientCaching //see https://redis.io/docs/latest/develop/reference/client-side-caching/
	//OptClientCachingTtl if OptClientCaching is true, then how long to keep the client side caching item. type: time.Duration
	OptClientCachingTtl
	//OptValkeyOptions Valkey client options. type: valkey.ClientOption
	OptValkeyOptions
	//OptManageTypes set true to manage data types. type: bool
	OptManageTypes
)
const (
	//ManagedDataTypeCacheKeyPrefix the prefix for the managed data type cache key.
	ManagedDataTypeCacheKeyPrefix = "gcm:"
	// ManagedDataTypeCacheTpl formatting string for the managed data type cache key.
	ManagedDataTypeCacheTpl = ManagedDataTypeCacheKeyPrefix + "%s"
)

func New(namespace string, host string, ttl time.Duration, clientCaching bool, clientCachingTtl time.Duration, manageTypes bool) storage.Storage {
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
		OptManageTypes: manageTypes,
	}

	adapter := new(adapter2.AbstractAdapter)
	adapter.Name = "valkey"
	adapter.SetOptions(opts)

	anyToString := func(v any) string {
		return fmt.Sprintf("%v", v)
	}

	setType := func(k string, v any) error {
		if !adapter.GetOptions()[OptManageTypes].(bool) {
			return nil
		}
		t := storage.GetType(v)
		if !adapter.GetOptions()[storage.OptDataTypes].(storage.DataTypes)[t] {
			return errs.Wrap(errors.ErrUnsupportedDataType, fmt.Sprintf("key: %s type: %d, value: %v", k, t, v))
		}
		cl := adapter.Client.(valkey.Client)
		key := fmt.Sprintf(ManagedDataTypeCacheTpl, adapter.NamespacedKey(k))
		_ = cl.Do(
			context.TODO(),
			cl.B().Set().Key(key).Value(anyToString(t)).Nx().Build(),
		)
		return nil
	}
	setTypeMulti := func(vals map[string]any) error {
		if !adapter.GetOptions()[OptManageTypes].(bool) {
			return nil
		}
		cl := adapter.Client.(valkey.Client)
		cmds := make(valkey.Commands, 0, len(vals))
		for k, v := range vals {
			t := storage.GetType(v)
			if !adapter.GetOptions()[storage.OptDataTypes].(storage.DataTypes)[t] {
				return errs.Wrap(errors.ErrUnsupportedDataType, fmt.Sprintf("key: %s type: %d, value: %v", k, t, v))
			}
			key := fmt.Sprintf(ManagedDataTypeCacheTpl, adapter.NamespacedKey(k))
			cmds = append(cmds, cl.B().Set().Key(key).Value(anyToString(t)).Nx().Build())
		}
		_ = cl.DoMulti(
			context.TODO(),
			cmds...,
		)
		return nil
	}

	getTyped := func(k, v string) (any, error) {
		if !adapter.GetOptions()[OptManageTypes].(bool) {
			return v, nil
		}
		cl := adapter.Client.(valkey.Client)
		key := fmt.Sprintf(ManagedDataTypeCacheTpl, adapter.NamespacedKey(k))
		resp := cl.Do(
			context.TODO(),
			cl.B().Get().Key(key).Build(),
		)
		if resp.Error() != nil {
			return nil, nil
		}
		tt, err := resp.ToString()
		if err != nil {
			return nil, errs.Wrap(err, "failed to get value")
		}
		t, err := strconv.Atoi(tt)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get type")
		}
		return storage.GetTypedValue(t, v)
	}

	getTypedMulti := func(vals map[string]any) (map[string]any, error) {
		if !adapter.GetOptions()[OptManageTypes].(bool) {
			return vals, nil
		}
		ret := make(map[string]any, len(vals))
		cl := adapter.Client.(valkey.Client)
		cmds := make(valkey.Commands, 0, len(vals))
		for k := range vals {
			cmds = append(cmds, cl.B().Get().Key(fmt.Sprintf(ManagedDataTypeCacheTpl, adapter.NamespacedKey(k))).Build().Pin())
		}
		resp := cl.DoMulti(
			context.TODO(),
			cmds...,
		)
		for i, r := range resp {
			cmdKey := adapter.StripNamespace(strings.Replace(cmds[i].Commands()[1], ManagedDataTypeCacheKeyPrefix, "", 1))
			tt, err := r.ToString()
			if err != nil {
				return nil, errs.Wrap(err, "failed to get value")
			}
			t, err := strconv.Atoi(tt)
			if err != nil {
				return nil, errs.Wrap(err, "failed to get type")
			}
			vv, err := storage.GetTypedValue(t, anyToString(vals[cmdKey]))
			if err != nil {
				return nil, errs.Wrap(err, "failed to get typed value")
			}
			ret[cmdKey] = vv
		}
		return ret, nil
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
				e := resp.Error()
				found = e == nil
				val, _ = resp.ToAny()
			}
			if !found {
				if adapter.GetChained() != nil {
					v, err2 := adapter.GetChained().GetItem(key)
					if err2 != nil {
						return nil, errors.ErrKeyNotFound
					}
					_, _ = adapter.SetItem(key, v)
					return getTyped(key, anyToString(v))
				}
				return nil, errors.ErrKeyNotFound
			}
			return getTyped(key, val.(string))
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
					cmds = append(
						cmds,
						valkey.CT(
							cl.B().Get().Key(nsKey).Cache().Pin(),
							adapter.GetOptions()[OptClientCachingTtl].(time.Duration),
						),
					)
				}
				for i, resp := range cl.DoMultiCache(context.TODO(), cmds...) {
					cmdKey := adapter.StripNamespace(cmds[i].Cmd.Commands()[1])
					if resp.Error() != nil {
						if adapter.GetChained() != nil {
							v, err3 := adapter.GetChained().GetItem(cmdKey)
							if err3 != nil {
								return ret, errs.Wrap(err3, "failed to get item")
							}
							adapter.SetItem(cmdKey, v)
							ret[cmdKey] = v
							continue
						}
						err2 = errs.Wrap(resp.Error(), "failed to get item")
						continue
					}
					v, err3 := resp.ToAny()
					ret[cmdKey] = v
					if err3 != nil {
						err2 = errs.Wrap(err3, "failed to get item")
					}
				}
			case false:
				//create the Valkey command set
				cmds := make(valkey.Commands, 0, len(keys))
				for _, key := range keys {
					nsKey := adapter.NamespacedKey(key)
					if !adapter.ValidateKey(nsKey) {
						return ret, errs.Wrap(errors.ErrKeyInvalid, "failed to get item")
					}
					cmds = append(cmds, cl.B().Get().Key(nsKey).Build().Pin())
				}
				for i, resp := range cl.DoMulti(context.TODO(), cmds...) {
					cmdKey := adapter.StripNamespace(cmds[i].Commands()[1])
					if resp.Error() != nil {
						//we only hit the chained cache one key at a time as response from this cache may have partial hits
						if adapter.GetChained() != nil {
							v, err3 := adapter.GetChained().GetItem(cmdKey)
							if err3 != nil {
								return ret, errs.Wrap(err3, "failed to get item")
							}
							adapter.SetItem(cmdKey, v)
							ret[cmdKey] = v
							continue
						}
						err2 = errs.Wrap(resp.Error(), "failed to get item")
						continue
					}
					v, err3 := resp.ToAny()
					ret[cmdKey] = v
					if err3 != nil {
						err2 = errs.Wrap(err3, "failed to get item")
					}
				}
			}
			if err2 != nil {
				return ret, err2
			}
			return getTypedMulti(ret)
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
			err3 := setType(key, value)
			if adapter.GetChained() != nil {
				_, _ = adapter.GetChained().SetItem(key, value)
			}
			return true, err3
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
				cmds = append(
					cmds,
					cl.B().Set().Key(nsKey).Value(anyToString(value)).Ex(adapter.GetOptions()[storage.OptTTL].(time.Duration)).Build().Pin(),
				)
			}

			for i, resp := range cl.DoMulti(context.TODO(), cmds...) {
				cmdKey := adapter.StripNamespace(cmds[i].Commands()[1])
				if resp.Error() != nil {
					err = errs.Wrap(resp.Error(), "failed to set item")
					continue
				}
				keys[i] = cmdKey
			}
			if err != nil {
				return keys, err
			}
			err3 := setTypeMulti(values)
			if adapter.GetChained() != nil {
				_, _ = adapter.GetChained().SetItems(values)
			}
			return keys, err3
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
