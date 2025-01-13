package storage

type Chainable interface {
	//ChainAdapter chains another adapter to this one. If this adapter does not have a value, the chained adapter will be processed
	ChainAdapter(adapter Storage) Storage
	//GetChained returns the chained adapter
	GetChained() Storage
}
