package errors

import "errors"

var ErrKeyNotFound  = errors.New("key not found")
var ErrKeyInvalid = errors.New("key invalid")
var ErrNotReadable = errors.New("not readable")
var ErrNotWritable = errors.New("not writable")
