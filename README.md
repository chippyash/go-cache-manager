# Multi Adapter Cache Manager
## github.com/chippyash/go-cache-manager

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
 - S3 Bucket

## How

Go V1.23.4+

**Please consider any version of this library at < 1 as pre production**. Use at your own risk. But please do try it out. The more feedback 
I get, the better it will be.  That said, I am using it in production.

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

The underlying client for the Memory Cache is [github.com/patrickmn/go-cache](https://github.com/patrickmn/go-cache)

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

As this can get tiresome if you have to do a lot of conversion, you can switch on data management by setting the 'manageTypes'
parameter for valkey.New() (the last parameter) to true.  When getting cache items, they will be returned as the same type 
in which they were set.

This is achieved by recording the item's data type in the cache in the 'gcm' namespace. Of course, having to hit the cache
twice, means that performance will be marginally impacted but you might consider this an acceptable cost in return for easier
data handling. It also means that the Valkey/Redis adapter works in the same way as the Memory adapter making swapping 
one for another a breeze.

```go
cacheManager := valkey.New(ns, host, ttl, clientCaching, clientCachingTtl, true).Open()
someValue := []byte("foobar")
ok, err := cacheManager.SetItem("key", someValue)
if !ok || err != nil {
	panic(errors.Wrap(err, "could not set value for key"))
}
v, err := cacheManager.GetItem("key")
if err != nil {
	panic(err)
}
fetchedValue := v.([]byte)
//true == bytes.Equal(someValue, fetchedValue)
```

For time.Time values to work correctly, we need to know the formatting string to use. This is set in the adapter
options (see 'Setting options' below). You need to set the `valkey.OptDatetimeFormat` to your required format. It is set to
`time.RFC3339` by default.

### S3 Bucket
This adapter is provided as a working example of how you can back your cache with an S3 bucket.  S3 provides cheap, but by
caching standards, slow storage. There are circumstances however that dictate that you want to have primary data stored in
S3, most usually in JSON or CSV format and use this as a source to feed more performant caches. As such, I wouldn't expect
this adapter to be the primary adapter, but chained behind another adapter. Hopefully, this adapter will guide you to 
roll your own file/object backing caches.

```go
import "github.com/chippyash/go-cache-manager/adapter/bucket"
import "github.com/aws/aws-sdk-go-v2/service/s3"

bucket := "my-cache-bucket"
prefix := "some/folder/"
suffix := ".json"
mimeType := bucket.MimeTypeJson
region := "eu-west-2"
cacheManager, err := bucket.New(bucket, prefix, suffix, mimeType, region)
if err != nil {
	panic(err)
}
```

It is limited in its functional ability:

 - It is only able to deal with `string` and `[]byte` input data types. Any other data type will produce an error.
 - It's functionality is limited as some methods don't make sense. Only the following methods are supported:
   - GetItem
   - GetItems
   - SetItem
   - SetItems
   - HasItem
   - HasItems
   - RemoveItem
   - RemoveItems
   - Open - No Op
   - Close - No Op
 - The Getter methods always return a string value

The `mimeType` parameter for the constructor is used by s3 to indicate the object type, so that it is properly parsed by other
S3 operations.

To unmarshal the value use something like:

```go
val, err := cacheManager.GetItem("key")
if err != nil {
	panic(err)
}
var obj MyObject
err := json.Unmarshal([]byte(val.(string)), &obj)
```

The most common usage for this type of cache is to store blobs of data, most commonly in JSON or CSV. (NB, I'm looking at 
extending this to Parquet and other formats).  For instance, in my shop, we have systems producing and updating some configuration
values (expressed as JSON) into an S3 bucket. The production application uses a Redis cache, behind a memory cache to store 
that config for fast access. But in the event of cache failure, it looks back to the S3 bucket as its source of truth.

![docs/nested-cache.puml](docs/nested-cache.png)

The underlying client for this adapter is the [AWS Go SDK V2 S3 Client]("github.com/aws/aws-sdk-go-v2/service/s3"). You 
are encouraged to RTFM on S3. Most problems using this adapter will be because you did not RTFM!

Authentication for the client uses the environment method based on environment variables etc.  See [AWS Config](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config)
for additional information.  If you need support for other methods, please consider a Pull Request.

### Namespaces
Each adapter allows you to declare a namespace. This is simply prefixed to any key value that you use. Thus, you can create multiple
cache adapters in your application and be certain that their entries are separated out in your cache backend.

Traditionally in Redis we split out cache names using the ':' character. This is recognised by many Redis clients and is 
used to create a tree hierarchy display of cache keys. To achieve the same set the namespace name as '\<name>:', e.g. 'categories:'.

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
import vk "github.com/chippyash/go-cache-manager/adapter/valkey"
import "github.com/valkey-io/valkey-go"

opts := cache.GetOptions()
valkeyOpts := opts[vk.OptValkeyOptions].(valkey.ClientOption)
//set up cluster connection
valkeyOpts.InitAddress = []string{"127.0.0.1:7001", "127.0.0.1:7002", "127.0.0.1:7003"}
valkeyOpts.ShuffleInit = true
opts[vk.OptValkeyOptions] = valkeyOpts
cache.SetOptions(opts)
cache, err := cache.Open()
```

### Using the underlying client
In some circumstances, this library may not give exactly what you want. In that case you can retrieve the underlying client
and act upon your cache backend more directly.

#### Memory Cache Client
```go
import "github.com/chippyash/go-cache-manager/adapter"
import "github.com/chippyash/go-cache-manager/adapter/memory"
import "github.com/patrickmn/go-cache"

cacheManager := memory.New(ns, ttl, purgeTtl)
client := cacheManager.(*adapter.AbstractAdapter).Client.(*cache.Cache)
```

#### Valkey Cache Client
```go
import "github.com/chippyash/go-cache-manager/adapter"
import vk "github.com/chippyash/go-cache-manager/adapter/valkey"
import "github.com/valkey-io/valkey-go"

cacheManager, err := vk.New(ns, host, ttl, clientCaching, clientCachingTtl, false).Open()
client := cacheManager.(*adapter.AbstractAdapter).Client.(valkey.Client)
```

#### S3 Bucket Client
```go
import "github.com/chippyash/go-cache-manager/adapter"
import "github.com/chippyash/go-cache-manager/adapter/bucket"
import "github.com/aws/aws-sdk-go-v2/service/s3"

cacheManager, err := bucket.New(bucket, prefix, suffix, mimeType, region)
client := cacheManager.(*adapter.AbstractAdapter).Client.(*s3.Client)
```

## For Development

If you want to add another adapter, you should carefully study the three so far provided.  Write your code, including the unit
tests, which as a simple base should mimic what are already done, plus any required by your specific adapter.

This lib is peculiar in that it follows a well trodden path, in the PHP world,  of using an Abstract parent to all adapters 
and then decorating the concrete objects with the functionality to carry out the interface methods.  Make sure you understand that. In effect 
an adapter is just a bunch of functions that are set when the adapter is constructed.  This makes it easy for you to:

 - Create an adapter on the fly
 - Amend an existing adapter to your specific requirements
#### Create an adapter on the fly
```go
import (
    "github.com/chippyash/go-cache-manager/adapter"
    "github.com/chippyash/go-cache-manager/storage"
)

adapter := new(adapter.AbstractAdapter)
//set the name
adapter.Name = "myadapter"
//set the client
adapter.Client = some.Client
//create minimal options and add them
opts := storage.StorageOptions{
    storage.OptNamespace:      "",
    storage.OptReadable:       true,
    storage.OptWritable:       true,
    storage.OptDataTypes:      storage.DefaultDataTypes,
}
adapter.SetOptions(opts)
//set just the functions you are actually going to use
adapter.
    SetGetItemFunc(func(key string) (any, error) { //your code here }).
    SetSetItemFunc(func(key string, value any) (bool, error) { //your code here })
//and use your adapter
a, _ := adapter.GetItem("key")
ok, _ := adapter.SetItem("anotherkey", "value")
```

#### Amend an existing adapter
Let's say, for example, that you don't quite like the way that the Valkey adapter handles a particular method, it doesn't
quite meet your requirements or that you have found a bug in the method, and you want to temporarily patch it until your pull 
request is merged.  Construct the adapter as normal, and then override the offending function:

```go
import (
    "github.com/chippyash/go-cache-manager/adapter"
    "github.com/chippyash/go-cache-manager/adapter/valkey"
)

adapter, _ := valkey.New(ns, host, ttl, clientCaching, clientCachingTtl, false).Open()
adapter.(adapter.AbstractAdapter).SetHasItemFunc(func(key string) bool { //your code here })
//then use the adapter as normal
```

### Changing this library
Changes to the existing interface will not be accepted without a long discussion because they may cause a BC break.

Additions to the interface can be accepted, as long as you make the changes to all the currently supported concrete 
implementations.

Updates to the documentation i.e. this README.md, are most welcome. I speak in UK English. That may not always make sense 
to our friends around the world, so if you have better ways of phrasing or want to provide another language README, then
please contact me, directly or via an issue in this repo.

As normal, fork the library, make your changes and request a pull request back into this repo. Put your changes on a branch.

### Unit Testing
`make test`

Unit tests for the Valkey adapter run against [miniredis](github.com/alicebob/miniredis/v2) which does not offer support
for client side caching. Thus, this piece of functionality is currently has no unit test, so be aware!  In due course, I'll get 
the test suite running against a real Valkey server. I do have it running against a production [Valkey](https://valkey.io/) 
server that supports client side caching and so far, no problems. It works out of the box. 

Unit tests for the S3 bucket run against a mock S3 client, so use with caution. As above, once I get this lib running against
a real S3 bucket via a third party test runner, you'll have more confidence.

## License
This software is released under the MIT License. See LICENSE.txt for details.

For license information of dependencies, please see licenses.csv.  If the dependencies change run `make license-check`
to update the file.

## Some thanks
 - Thanks to the original Zend team that came up with Zend/Cache, some of whom I know, but they shall be nameless. They
gave many hours of sport back in the day!
 - Thanks to [@rueian](https://github.com/rueian) for his very fast support on Valkey. He is a main contributor to Valkey.
Thanks to him the initial build of this lib (memory, valkey) took 2.5 days to write instead of 2.5 weeks.