package storage

const (
	TypeNull = iota
	TypeBoolean
	TypeInteger
	TypeInteger32
	TypeInteger64
	TypeFloat
	TypeFloat64
	TypeString
)

const (
	UnknownType = "UnknownType"
)

const (
	OptNamespace = iota
	OptKeyPattern
	OptReadable
	OptWritable
	OptTTL
	OptMaxKeyLength   //future use
	OptMaxValueLength //future use
	OptDataTypes      //future use
)

type StorageOptions map[int]any
type DataTypes map[int]bool

type Storage interface {
	//SetOptions sets the storage options
	SetOptions(opts StorageOptions)
	//GetOptions returns the storage options
	GetOptions() StorageOptions
	//GetItem returns the value for stored item identified by key
	GetItem(key string) (any, error)
	//GetItems sets multiple values.
	GetItems(keys []string) (map[string]any, error)
	//HasItem returns true if storage has the requested key, else false
	HasItem(key string) bool
	//HasItems returns a keyed array of bools denoting if required keys are in the storage
	HasItems(keys []string) map[string]bool
	//SetItem sets the value of the requested key. Returns true if set, else false and a possible error
	SetItem(key string, value any) (bool, error)
	//SetItems sets multiple key values. Returns an array of keys not set and a possible error
	SetItems(values map[string]any) ([]string, error)
	//CheckAndSetItem sets the value of the requested key if the key already exists. Returns true if set, else false and a possible error
	CheckAndSetItem(key string, value any) (bool, error)
	//CheckAndSetItems sets the value of multiple requested keys if the keys already exist. Returns an array of keys not set and a possible error
	CheckAndSetItems(values map[string]any) ([]string, error)
	//TouchItem resets the TTL for the given key. Returns true if reset, else false
	TouchItem(key string) bool
	//TouchItems resets the TTL for multiple keys. Returns an array of keys not reset
	TouchItems(keys []string) []string
	//RemoveItem removes (deletes) the requested key. Returns true if removed, else false
	RemoveItem(key string) bool
	//RemoveItems removes (deletes) multiple key. Returns array of keys not removed
	RemoveItems(keys []string) []string
	//Increment increments the key value by n. If the key is none numeric or not found, an error will be returned
	Increment(key string, n int64) (int64, error)
	//Decrement decrements the key value by n. If the key is none numeric or not found, an error will be returned
	Decrement(key string, n int64) (int64, error)
	//Open opens or starts the adapter
	Open() (Storage, error)
	//Close closes down the adapter
	Close() error
}

var DefaultDataTypes = DataTypes{
	TypeNull:      true,
	TypeBoolean:   true,
	TypeInteger:   true,
	TypeInteger32: true,
	TypeInteger64: true,
	TypeFloat:     true,
	TypeFloat64:   true,
	TypeString:    true,
}
