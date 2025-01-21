package errors

import "github.com/pkg/errors"

var ErrKeyNotFound  = errors.New("key not found")
var ErrKeyInvalid = errors.New("key invalid")
var ErrNotReadable = errors.New("not readable")
var ErrNotWritable = errors.New("not writable")
var ErrUnsupportedDataType = errors.New("unsupported data type")
var ErrNotImplemented = errors.New("not implemented")