package adapter

import (
	"github.com/chippyash/go-cache-manager/storage"
	"regexp"
	"strings"
)

// AbstractAdapter is the abstract base adapter on which all adapters are built.
// It implements the Storage interface
type AbstractAdapter struct {
	Name             string
	Client           any
	chained          storage.Storage
	options          storage.StorageOptions
	getItem          func(key string) (any, error)
	getItems         func(keys []string) (map[string]any, error)
	hasItem          func(key string) bool
	hasItems         func(keys []string) map[string]bool
	setItem          func(key string, value any) (bool, error)
	setItems         func(values map[string]any) ([]string, error)
	checkAndSetItem  func(key string, value any) (bool, error)
	checkAndSetItems func(values map[string]any) ([]string, error)
	touchItem        func(key string) bool
	touchItems       func(keys []string) []string
	removeItem       func(key string) bool
	removeItems      func(keys []string) []string
	increment        func(key string, n int64) (int64, error)
	decrement        func(key string, n int64) (int64, error)
	open             func() (storage.Storage, error)
	close            func() error
}

/** Storage Interface **/

func (a *AbstractAdapter) SetOptions(opts storage.StorageOptions) {
	a.options = opts
}

func (a *AbstractAdapter) GetOptions() storage.StorageOptions {
	return a.options
}

func (a *AbstractAdapter) GetItem(key string) (any, error) {
	return a.getItem(key)
}

func (a *AbstractAdapter) GetItems(keys []string) (map[string]any, error) {
	return a.getItems(keys)
}

func (a *AbstractAdapter) HasItem(key string) bool {
	return a.hasItem(key)
}

func (a *AbstractAdapter) HasItems(keys []string) map[string]bool {
	return a.hasItems(keys)
}

func (a *AbstractAdapter) SetItem(key string, value any) (bool, error) {
	return a.setItem(key, value)
}

func (a *AbstractAdapter) SetItems(values map[string]any) ([]string, error) {
	return a.setItems(values)
}

func (a *AbstractAdapter) CheckAndSetItem(key string, value any) (bool, error) {
	return a.checkAndSetItem(key, value)
}

func (a *AbstractAdapter) CheckAndSetItems(values map[string]any) ([]string, error) {
	return a.checkAndSetItems(values)
}

func (a *AbstractAdapter) TouchItem(key string) bool {
	return a.touchItem(key)
}

func (a *AbstractAdapter) TouchItems(keys []string) []string {
	return a.touchItems(keys)
}

func (a *AbstractAdapter) RemoveItem(key string) bool {
	return a.removeItem(key)
}

func (a *AbstractAdapter) RemoveItems(keys []string) []string {
	return a.removeItems(keys)
}

func (a *AbstractAdapter) Increment(key string, n int64) (int64, error) {
	return a.increment(key, n)
}

func (a *AbstractAdapter) Decrement(key string, n int64) (int64, error) {
	return a.decrement(key, n)
}

/** Chainable Interface **/

func (a *AbstractAdapter) ChainAdapter(adapter storage.Storage) storage.Storage {
	a.chained = adapter
	return a
}

func (a *AbstractAdapter) GetChained() storage.Storage {
	return a.chained
}

func (a *AbstractAdapter) Open() (storage.Storage, error) {
	return a.open()
}

func (a *AbstractAdapter) Close() error {
	return a.close()
}

/** Utility functions **/

// NamespacedKey returns the key suffixed with namespace if any
func (a *AbstractAdapter) NamespacedKey(key string) string {
	ns := a.options[storage.OptNamespace].(string)
	if ns != "" {
		key = ns + key
	}
	return key
}

func (a *AbstractAdapter) StripNamespace(key string) string {
	ns := a.options[storage.OptNamespace].(string)
	if ns != "" {
		key = strings.Replace(key, ns, "", 1)
	}
	return key
}

// ValidateKey validates the key against the regex pattern in options[storage.OptKeyPattern] if any
func (a *AbstractAdapter) ValidateKey(key string) bool {
	p := a.options[storage.OptKeyPattern].(string)
	if p == "" {
		return true
	}
	re := regexp.MustCompile(p)
	return re.MatchString(key)
}

/** Setters for the Storage interface functions **/

func (a *AbstractAdapter) SetGetItemFunc(f func(key string) (any, error)) *AbstractAdapter {
	a.getItem = f
	return a
}

func (a *AbstractAdapter) SetGetItemsFunc(f func(keys []string) (map[string]any, error)) *AbstractAdapter {
	a.getItems = f
	return a
}

func (a *AbstractAdapter) SetSetItemFunc(f func(key string, value any) (bool, error)) *AbstractAdapter {
	a.setItem = f
	return a
}

func (a *AbstractAdapter) SetSetItemsFunc(f func(values map[string]any) ([]string, error)) *AbstractAdapter {
	a.setItems = f
	return a
}

func (a *AbstractAdapter) SetHasItemFunc(f func(key string) bool) *AbstractAdapter {
	a.hasItem = f
	return a
}

func (a *AbstractAdapter) SetHasItemsFunc(f func(keys []string) map[string]bool) *AbstractAdapter {
	a.hasItems = f
	return a
}

func (a *AbstractAdapter) SetCheckAndSetItemFunc(f func(key string, value any) (bool, error)) *AbstractAdapter {
	a.checkAndSetItem = f
	return a
}

func (a *AbstractAdapter) SetCheckAndSetItemsFunc(f func(values map[string]any) ([]string, error)) *AbstractAdapter {
	a.checkAndSetItems = f
	return a
}

func (a *AbstractAdapter) SetTouchItemFunc(f func(key string) bool) *AbstractAdapter {
	a.touchItem = f
	return a
}

func (a *AbstractAdapter) SetTouchItemsFunc(f func(keys []string) []string) *AbstractAdapter {
	a.touchItems = f
	return a
}

func (a *AbstractAdapter) SetRemoveItemFunc(f func(key string) bool) *AbstractAdapter {
	a.removeItem = f
	return a
}

func (a *AbstractAdapter) SetRemoveItemsFunc(f func(keys []string) []string) *AbstractAdapter {
	a.removeItems = f
	return a
}

func (a *AbstractAdapter) SetIncrementFunc(f func(key string, n int64) (int64, error)) *AbstractAdapter {
	a.increment = f
	return a
}

func (a *AbstractAdapter) SetDecrementFunc(f func(key string, n int64) (int64, error)) *AbstractAdapter {
	a.decrement = f
	return a
}

func (a *AbstractAdapter) SetOpenFunc(f func() (storage.Storage, error)) *AbstractAdapter {
	a.open = f
	return a
}

func (a *AbstractAdapter) SetCloseFunc(f func() error) *AbstractAdapter {
	a.close = f
	return a
}
