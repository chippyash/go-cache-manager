# Multi Adapter Cache Manager
## chippash/go-cache-manager

## What

Provides a simplified interface to key/value cache management utilising adaptors to provide the same functionality for
different cache databases.

### Current backends

 - Memory
 - Valkey/Redis

## How

## For Production

```go
import "github.com/chippyash/go-cache-manager/adapter"
import "github.com/chippyash/go-cache-manager/storage"
```

### Memory Cache

```go
ns := "myNameSpace:"  //this can be left blank
ttl := time.Minute * 5 //Expiry Ttl
purgeTtl := time.Minute * 10  //set the purgeTtl > ttl
cacheManager := adapter.memory.New(ns, ttl, purgeTtl)
```

The underlying client for the Memory Cache is [github.com/patrickmn/go-cache](github.com/patrickmn/go-cache)

### Valkey (Redis) Cache

```go
ns := "myNameSpace:"  //this can be left blank
host := "127.0.0.1"
ttl := time.Minute * 5 //Expiry Ttl
clientCaching := true //we want to use client side caching
clientCachingTtl = time.Minute * 4 //set this to less than the ttl
cacheManager, err := adapter.valkey.New(ns, host, ttl, clientCaching, clientCachingTtl).Open()
if err != nil {
	panic(err)
}
```

The underlying client for the Valkey Cache is [github.com/valkey-io/valkey-go](github.com/valkey-io/valkey-go)

### Namespaces
Each adapter allows you to declare a namespace. This is simply prefixed to any key value that you use. Thus, you can create multiple
cache adapters in your application and be certain that their entries are separated out in your cache backend.

### Chaining adapters
The library supports chaining adapters together.

```go
cacheManager := adapter.memory.New(ns, ttl, purgeTtl)
chainedAdapter, err := := adapter.valkey.New(ns, host, ttl, false, time.Second * 0).Open()
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
adapter, err := adapter.valkey.New(ns, host, ttl, false, time.Second * 0).Open()
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
cache := adapter.valkey.New(ns, host, ttl, false, time.Second * 0)
opts := cache.GetOptions()
//do something with the options
// ...
cache.SetOptions(opts)
cache, err := cache.Open()
```

Note that the options are untyped. You need to type them correctly in order to use them.

```go
opts := cache.GetOptions()
valkeyOpts := opts[adapter.valkey.OptValkeyOptions].(valkey.ClientOption)
//set up cluster connection
valkeyOpts.InitAddress = []string{"127.0.0.1:7001", "127.0.0.1:7002", "127.0.0.1:7003"}
valkeyOpts.ShuffleInit = true
opts[adapter.valkey.OptValkeyOptions] = valkeyOpts
cache.SetOptions(opts)
cache, err := cache.Open()
```

### Using the underlying client
In some circumstances, this library may not give exactly what you want. In that case you can retrieve the underlying client
and act upon your cache backend more directly.

#### Memory Cache Client
```go
import "github.com/chippyash/go-cache-manager/adapter"
import "github.com/patrickmn/go-cache"

cacheManager := adapter.memory.New(ns, ttl, purgeTtl)
client := cacheManager.Client.(*cache.Cache)
```

#### Valkey Cache Client
```go
import "github.com/chippyash/go-cache-manager/adapter"
import "github.com/valkey-io/valkey-go"

cacheManager, err := adapter.valkey.New(ns, host, ttl, clientCaching, clientCachingTtl).Open()
client := cacheManager.Client.(valkey.Client)
```

## For Development

### Unit Testing
`make test`

## License
This software is released under the MIT License. See LICENSE.txt for details.

For license information of dependencies, please see licenses.csv.  If the dependencies change run `make license-check`
to update the file.
