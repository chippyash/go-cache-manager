# Multi Adapter Cache Manager
## chippash/go-cache-manager

## What

Provides a simplified interface to key/value cache management utilising adaptors to provide the same functionality for
different cache databases.

Based around some ideas that come from my PHP days and particularly the Zend Framework (now Laminas) that had a fully 
functional simple cache manager. See https://docs.laminas.dev/laminas-cache/

This will never try to be a friend to all. For straightforward caching operations, it should work.

### Design considerations

 - Given the Go language, be as simple as possible
 - Deal with straight forward key/value caching
 - Make each cache backend look the same, so they can be swapped in and out
 - Allow devs to use the native client if they need to 

### Current backends

 - Memory
 - Valkey/Redis

## How

Go V1.23.4+

Please consider any version at < 1 as pre production. Use at your own risk. But please do try it out. The more feedback 
I get, the better it will be.

## For Production

```go
import "github.com/chippyash/go-cache-manager/adapter/valkey"
import "github.com/chippyash/go-cache-manager/adapter/memory"
import "github.com/chippyash/go-cache-manager/storage"
```

### Memory Cache

```go
ns := "myNameSpace:"  //this can be left blank
ttl := time.Minute * 5 //Expiry Ttl
purgeTtl := time.Minute * 10  //set the purgeTtl > ttl
cacheManager := memory.New(ns, ttl, purgeTtl)
```

The underlying client for the Memory Cache is [github.com/patrickmn/go-cache](github.com/patrickmn/go-cache)

Using the setter methods for the Memory cache adapter allows you pass any type of value. The value will be stored as-is
in memory and returned via any of the getter methods as type interface{} (any). You are responsible for casting it to your 
required type;

```go
cacheManager := memory.New(ns, ttl, purgeTtl)
ok, err := cacheManager.SetItem("key", someValue)
if !ok || err != nil {
	panic(errors.Wrap(err, "could not set value for key"))
}
v, err := cacheManager.GetItem("key")
if err != nil {
	panic(err)
}
someValue := v.(uint64)
```
### Valkey (Redis) Cache

```go
ns := "myNameSpace:"  //this can be left blank
host := "127.0.0.1"
ttl := time.Minute * 5 //Expiry Ttl
clientCaching := true //we want to use client side caching
clientCachingTtl = time.Minute * 4 //set this to less than the ttl
cacheManager, err := valkey.New(ns, host, ttl, clientCaching, clientCachingTtl, false).Open()
if err != nil {
	panic(err)
}
```

The underlying client for the Valkey Cache is [github.com/valkey-io/valkey-go](github.com/valkey-io/valkey-go)

Using the setter methods for the Valkey cache adapter allows you pass any type of value. The value will be stored as a
string in the cache server and returned via any of the getter methods as type **interface{}|string**. You are responsible for casting it 
to your required type.

```go
cacheManager := valkey.New(ns, host, ttl, clientCaching, clientCachingTtl, false).Open()
ok, err := cacheManager.SetItem("key", someValue)
if !ok || err != nil {
	panic(errors.Wrap(err, "could not set value for key"))
}
v, err := cacheManager.GetItem("key")
if err != nil {
	panic(err)
}
someValue, err := strconv.ParseUint(v.(string), 10, 64)
if err != nil {
    panic(err)
}
//note that for other types, you may need a second cast
u64, err := strconv.ParseUint(v.(string), 10, 32)
if err != nil {
    panic(err)
}
someValue = uint32(u64)
```

### Namespaces
Each adapter allows you to declare a namespace. This is simply prefixed to any key value that you use. Thus, you can create multiple
cache adapters in your application and be certain that their entries are separated out in your cache backend.

### Chaining adapters
The library supports chaining adapters together.

```go
cacheManager := memory.New(ns, ttl, purgeTtl)
chainedAdapter, err := := valkey.New(ns, host, ttl, false, time.Second * 0, false).Open()
cacheManager.(storage.Chainable).ChainAdapter(chainedAdapter)
```

This is a solution that allows you to put a memory cache in front of a Valkey/Redis cache.  Note however, that if your
Valkey/Redis server supports client side caching, you can simply use the previous example for 'Valkey (Redis) Cache'

### Adapter Methods
For a full list of available adapter methods (functions) see [the Storage interface](storage/storageinterface.go)

#### The Open and Close methods
Some adapters will try to make an immediate connection to the data source when they are constructed. The Valkey adapter
is one such adapter. For this reason we delay the connection until the Open() method is called. This allows you to construct
the adapter in advance, maybe in your DIC, and then open it when you actually need it.  The memory adapter doesn't need you to 
call Open, but as a matter of course, you should do as this allows for greater interchangeability between adapters.

Similarly, although no functionality currently exists in the Close method for the provided adapters, you should get into
the habit of deferring a call to it.

```go
adapter, err := valkey.New(ns, host, ttl, false, time.Second * 0, false).Open()
if err != nil {
	panic(err)
}
defer(adapter.Close())
//or more properly
defer(func(){
	if err := adapter.Close(); err != nil {
        panic(err)
    }
})
```

### Setting options
Each adapter is set with a sane set of options.  You can reconfigure the options just after instantiating the adapter
by calling `adapter.SetOptions(opts)`. opts is a storage.StorageOptions object. Note that each adapter will have a default
set of options, but may have additional options specific to the adapter.

```go
cache := valkey.New(ns, host, ttl, false, time.Second * 0, false)
opts := cache.GetOptions()
//do something with the options
// ...
cache.SetOptions(opts)
cache, err := cache.Open()
```

Note that the options are untyped. You need to type them correctly in order to use them.

```go
opts := cache.GetOptions()
valkeyOpts := opts[valkey.OptValkeyOptions].(valkey.ClientOption)
//set up cluster connection
valkeyOpts.InitAddress = []string{"127.0.0.1:7001", "127.0.0.1:7002", "127.0.0.1:7003"}
valkeyOpts.ShuffleInit = true
opts[valkey.OptValkeyOptions] = valkeyOpts
cache.SetOptions(opts)
cache, err := cache.Open()
```

### Using the underlying client
In some circumstances, this library may not give exactly what you want. In that case you can retrieve the underlying client
and act upon your cache backend more directly.

#### Memory Cache Client
```go
import "github.com/chippyash/go-cache-manager/adapter/memory"
import "github.com/patrickmn/go-cache"

cacheManager := memory.New(ns, ttl, purgeTtl)
client := cacheManager.Client.(*cache.Cache)
```

#### Valkey Cache Client
```go
import vk "github.com/chippyash/go-cache-manager/adapter/valkey"
import "github.com/valkey-io/valkey-go"

cacheManager, err := vk.New(ns, host, ttl, clientCaching, clientCachingTtl, false).Open()
client := cacheManager.Client.(valkey.Client)
```

## For Development

If you want to add another adapter, you should carefully study the two so far provided.  Write your code, including the unit
tests, which as a simple base should mimic what are already done, plus any required by your specific adapter.

This lib is peculiar in that it follows a well trodden path of using an Abstract parent to all adapters and then decorates
it with the functionality to carry out the interface methods.  Make sure you understand that.

Changes to the existing interface will not be accepted without a long discussion because they may cause a BC break.

Additions to the interface can be accepted, as long as you make the changes to all the currently supported concrete 
implementations.

As normal, fork the library, make your changes and request a pull request back into this repo. Put your changes on a branch.

### Unit Testing
`make test`

## License
This software is released under the MIT License. See LICENSE.txt for details.

For license information of dependencies, please see licenses.csv.  If the dependencies change run `make license-check`
to update the file.
