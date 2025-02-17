package bucket

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	adapter2 "github.com/chippyash/go-cache-manager/adapter"
	"github.com/chippyash/go-cache-manager/errors"
	"github.com/chippyash/go-cache-manager/storage"
	errs "github.com/pkg/errors"
	"io"
)

const (
	//OptS3Bucket s3 bucket name
	OptS3Bucket = iota + storage.OptDataTypes + 1
	//OptS3Suffix object key suffix e.g. '.json'
	OptS3Suffix
	//OptS3MimeType object mime type e.g. 'application/json'. Use MimeTypeJson or MimeTypeText
	OptS3MimeType
	//OptS3Region AWS region e.g. 'eu-west-2'
	OptS3Region
)
const (
	MimeTypeJson = "application/json"
	MimeTypeText = "text/plain"
)

func New(bucket string, prefix string, suffix string, mimeType string, region string) (storage.Storage, error) {
	//set the options
	dTypes := storage.DataTypes{
		storage.TypeUnknown:   false,
		storage.TypeBoolean:   false,
		storage.TypeInteger:   false,
		storage.TypeInteger8:  false,
		storage.TypeInteger16: false,
		storage.TypeInteger32: false,
		storage.TypeInteger64: false,
		storage.TypeUint:      false,
		storage.TypeUint8:     true,
		storage.TypeUint16:    false,
		storage.TypeUint32:    false,
		storage.TypeUint64:    false,
		storage.TypeFloat32:   false,
		storage.TypeFloat64:   false,
		storage.TypeString:    true,
		storage.TypeDuration:  false,
		storage.TypeTime:      false,
		storage.TypeBytes:     false,
	}

	opts := storage.StorageOptions{
		storage.OptNamespace:      prefix,
		storage.OptKeyPattern:     "",
		storage.OptReadable:       true,
		storage.OptWritable:       true,
		storage.OptMaxKeyLength:   0,
		storage.OptMaxValueLength: 0,
		storage.OptDataTypes:      dTypes,
		OptS3Bucket:               bucket,
		OptS3Suffix:               suffix,
		OptS3MimeType:             mimeType,
		OptS3Region:               region,
	}

	//aws setup
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, errs.Wrap(err, "failed to load aws config")
	}

	adapter := new(adapter2.AbstractAdapter)
	adapter.Name = "s3"
	adapter.Client = s3.NewFromConfig(awsConfig)
	adapter.SetOptions(opts)

	//isString := func(mimeType string) bool {
	//	return strings.HasPrefix(mimeType, "text")
	//}

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
			nsKey = nsKey + adapter.GetOptions()[OptS3Suffix].(string)
			bckt := adapter.GetOptions()[OptS3Bucket].(string)
			input := &s3.GetObjectInput{
				Bucket: &bckt,
				Key:    &nsKey,
			}
			out, err := adapter.Client.(S3Iface).GetObject(context.TODO(), input)
			if err != nil {
				if adapter.GetChained() != nil {
					val, err := adapter.GetChained().GetItem(key)
					if err != nil {
						return nil, errors.ErrKeyNotFound
					}
					_, e := adapter.SetItem(key, val)
					return val, e
				}
				return nil, errs.Wrap(errors.ErrKeyNotFound, err.Error())
			}

			ret, err := io.ReadAll(out.Body)
			return string(ret), err
		}).
		SetGetItemsFunc(func(keys []string) (map[string]any, error) {
			ret := make(map[string]any)
			var err error
			for _, key := range keys {
				val, found := adapter.GetItem(key)
				if found != nil {
					if errs.Is(found, errors.ErrKeyInvalid) || errs.Is(found, errors.ErrNotReadable) {
						err = found
						continue
					}
					err = errors.ErrKeyNotFound
					continue
				}
				ret[key] = val
			}
			return ret, err
		}).
		SetSetItemFunc(func(key string, value any) (bool, error) {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false, errors.ErrNotWritable
			}
			t := storage.GetType(value)
			if !adapter.GetOptions()[storage.OptDataTypes].(storage.DataTypes)[t] {
				return false, errs.Wrap(errors.ErrUnsupportedDataType, fmt.Sprintf("key: %s type: %d, value: %v", key, t, value))
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false, errors.ErrKeyInvalid
			}
			nsKey = nsKey + adapter.GetOptions()[OptS3Suffix].(string)
			bckt := adapter.GetOptions()[OptS3Bucket].(string)
			mtype := adapter.GetOptions()[OptS3MimeType].(string)
			//convert value []byte dependent on its actual type
			var v []byte
			if t == storage.TypeString {
				v = []byte(value.(string))
			} else {
				v = value.([]byte)
			}
			input := &s3.PutObjectInput{
				Bucket:      &bckt,
				Key:         &nsKey,
				Body:        bytes.NewReader(v),
				ContentType: &mtype,
			}
			_, err := adapter.Client.(S3Iface).PutObject(context.TODO(), input)
			if err != nil {
				return false, errs.Wrap(err, "failed to put object")
			}
			if adapter.GetChained() != nil {
				_, _ = adapter.GetChained().SetItem(key, value)
			}
			return true, nil
		}).
		SetSetItemsFunc(func(values map[string]any) ([]string, error) {
			keys := make([]string, len(values))
			i := 0
			var err error
			for key, value := range values {
				_, e := adapter.SetItem(key, value)
				if e != nil {
					err = e
				}
				keys[i] = key
				i++
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
			nsKey = nsKey + adapter.GetOptions()[OptS3Suffix].(string)
			bckt := adapter.GetOptions()[OptS3Bucket].(string)
			input := &s3.HeadObjectInput{
				Bucket: &bckt,
				Key:    &nsKey,
			}
			_, err := adapter.Client.(S3Iface).HeadObject(context.TODO(), input)
			if err != nil {
				if adapter.GetChained() != nil {
					return adapter.GetChained().HasItem(key)
				}
				return false
			}

			return true
		}).
		SetHasItemsFunc(func(keys []string) map[string]bool {
			ret := make(map[string]bool)
			for _, key := range keys {
				ret[key] = adapter.HasItem(key)
			}
			return ret
		}).
		SetCheckAndSetItemFunc(func(key string, value any) (bool, error) {
			return false, errors.ErrNotImplemented
		}).
		SetCheckAndSetItemsFunc(func(values map[string]any) ([]string, error) {
			return make([]string, 0), errors.ErrNotImplemented
		}).
		SetTouchItemFunc(func(key string) bool {
			//errors.ErrNotImplemented
			return false
		}).
		SetTouchItemsFunc(func(keys []string) []string {
			//errors.ErrNotImplemented
			return make([]string, 0)
		}).
		SetRemoveItemFunc(func(key string) bool {
			if !adapter.GetOptions()[storage.OptWritable].(bool) {
				return false
			}
			nsKey := adapter.NamespacedKey(key)
			if !adapter.ValidateKey(nsKey) {
				return false
			}
			nsKey = nsKey + adapter.GetOptions()[OptS3Suffix].(string)
			bckt := adapter.GetOptions()[OptS3Bucket].(string)
			input := &s3.DeleteObjectInput{
				Bucket: &bckt,
				Key:    &nsKey,
			}
			out, err := adapter.Client.(S3Iface).DeleteObject(context.TODO(), input)
			if adapter.GetChained() != nil {
				return adapter.GetChained().RemoveItem(key)
			}
			return *out.DeleteMarker && err == nil
		}).
		SetRemoveItemsFunc(func(keys []string) []string {
			for _, key := range keys {
				adapter.RemoveItem(key)
			}
			return keys
		}).
		SetIncrementFunc(func(key string, n int64) (int64, error) {
			return 0, errors.ErrNotImplemented
		}).
		SetDecrementFunc(func(key string, n int64) (int64, error) {
			return 0, errors.ErrNotImplemented
		}).
		SetOpenFunc(func() (storage.Storage, error) {
			return adapter, nil
		}).
		SetCloseFunc(func() error {
			return nil
		})

	return adapter, nil
}

// S3Iface provides an interface to AWS SDK V2 s3 client. We require this for mocking the client in tests
type S3Iface interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}
