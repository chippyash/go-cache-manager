package bucket_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chippyash/go-cache-manager/adapter"
	"github.com/chippyash/go-cache-manager/adapter/bucket"
	"github.com/chippyash/go-cache-manager/adapter/memory"
	"github.com/chippyash/go-cache-manager/errors"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"maps"
	"slices"
	"testing"
	"time"
)

type MockS3Client struct {
	mock.Mock
	bucket.S3Iface
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(options *s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)

	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(options *s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func (m *MockS3Client) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.DeleteObjectsOutput), args.Error(1)
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func TestS3Adapter_GetAndSetItem(t *testing.T) {
	sut, err := bucket.New("testbucket", "/folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)
	mockS3 := new(MockS3Client)
	sut.(*adapter.AbstractAdapter).Client = mockS3
	setInput := &s3.PutObjectInput{
		Bucket:      aws.String("testbucket"),
		Key:         aws.String("/folder/key.json"),
		Body:        bytes.NewReader([]byte(`{"value":"foo"}`)),
		ContentType: aws.String(bucket.MimeTypeJson),
	}
	setOutput := &s3.PutObjectOutput{}
	mockS3.On("PutObject", context.TODO(), setInput).Return(setOutput, nil)
	ok, err := sut.SetItem("key", `{"value":"foo"}`)
	assert.True(t, ok)
	assert.NoError(t, err)

	getInput := &s3.GetObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("/folder/key.json"),
	}
	getOutput := &s3.GetObjectOutput{
		Body:        io.NopCloser(bytes.NewReader([]byte(`{"value":"foo"}`))),
		ContentType: aws.String(bucket.MimeTypeJson),
	}
	mockS3.On("GetObject", context.TODO(), getInput).Return(getOutput, nil)
	val, err := sut.GetItem("key")
	assert.NoError(t, err)
	type MyStruct struct {
		Value string `json:"value"`
	}
	var obj MyStruct
	err = json.Unmarshal([]byte(val.(string)), &obj)
	assert.NoError(t, err)
	assert.Equal(t, "foo", obj.Value)

	mockS3.AssertExpectations(t)
}

func TestS3Adapter_GetUnknownItem(t *testing.T) {
	sut, _ := bucket.New("testbucket", "/folder/", ".txt", bucket.MimeTypeText, "eu-west-2")
	mockS3 := new(MockS3Client)
	sut.(*adapter.AbstractAdapter).Client = mockS3
	getInput := &s3.GetObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("/folder/key.txt"),
	}
	getOutput := &s3.GetObjectOutput{
		Body:        io.NopCloser(bytes.NewReader([]byte(``))),
		ContentType: aws.String(bucket.MimeTypeJson),
	}
	mockS3.On("GetObject", context.TODO(), getInput).Return(getOutput, errs.New("s3 error"))

	val, err := sut.GetItem("key")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
	assert.Equal(t, nil, val)

	mockS3.AssertExpectations(t)
}

func TestS3Adapter_GetAndSetMultipleItems(t *testing.T) {
	sut, _ := bucket.New("testbucket", "/folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	mockS3 := new(MockS3Client)
	sut.(*adapter.AbstractAdapter).Client = mockS3
	vals := map[string]any{
		"TypeString": `{"value":"foo"}`,
	}

	setInput := &s3.PutObjectInput{
		Bucket:      aws.String("testbucket"),
		Key:         aws.String("/folder/TypeString.json"),
		Body:        bytes.NewReader([]byte(`{"value":"foo"}`)),
		ContentType: aws.String(bucket.MimeTypeJson),
	}
	setOutput := &s3.PutObjectOutput{}
	mockS3.On("PutObject", context.TODO(), setInput).Return(setOutput, nil)

	keys, err := sut.SetItems(vals)
	assert.NoError(t, err)
	expectedKeys := slices.Collect(maps.Keys(vals))
	assert.ElementsMatch(t, expectedKeys, keys)

	getInput := &s3.GetObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("/folder/TypeString.json"),
	}
	getOutput := &s3.GetObjectOutput{
		Body:        io.NopCloser(bytes.NewReader([]byte(`{"value":"foo"}`))),
		ContentType: aws.String(bucket.MimeTypeJson),
	}
	mockS3.On("GetObject", context.TODO(), getInput).Return(getOutput, nil)

	ret, err := sut.GetItems(keys)
	assert.NoError(t, err)
	for k, v := range vals {
		assert.Equal(t, v.(string), ret[k].(string))
	}

	mockS3.AssertExpectations(t)
}

func TestS3Adapter_HasItem(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)
	mockS3 := new(MockS3Client)
	sut.(*adapter.AbstractAdapter).Client = mockS3

	headInput1 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/key.json"),
	}
	headInput2 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/notfound.json"),
	}
	headOutput := &s3.HeadObjectOutput{}

	//object found test
	mockS3.On("HeadObject", context.TODO(), headInput1).Return(headOutput, nil)
	assert.True(t, sut.HasItem("key"))

	//not found test
	mockS3.On("HeadObject", context.TODO(), headInput2).Return(headOutput, errs.New("s3 error"))
	assert.False(t, sut.HasItem("notfound"))

	mockS3.AssertExpectations(t)
}

func TestS3Adapter_HasMultipleItems(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)
	mockS3 := new(MockS3Client)
	sut.(*adapter.AbstractAdapter).Client = mockS3

	headInput1 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/key1.json"),
	}
	headInput2 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/key2.json"),
	}
	headInput3 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/key3.json"),
	}
	headInput4 := &s3.HeadObjectInput{
		Bucket: aws.String("testbucket"),
		Key:    aws.String("folder/notfound.json"),
	}
	headOutput := &s3.HeadObjectOutput{}

	mockS3.On("HeadObject", context.TODO(), headInput1).Return(headOutput, nil)
	mockS3.On("HeadObject", context.TODO(), headInput2).Return(headOutput, nil)
	mockS3.On("HeadObject", context.TODO(), headInput3).Return(headOutput, nil)
	mockS3.On("HeadObject", context.TODO(), headInput4).Return(headOutput, errs.New("s3 error"))

	ret := sut.HasItems([]string{"key1", "key2", "key3", "notfound"})
	assert.True(t, ret["key1"])
	assert.True(t, ret["key2"])
	assert.True(t, ret["key3"])
	assert.False(t, ret["notfound"])

	mockS3.AssertExpectations(t)
}

//func TestS3Adapter_Chaining(t *testing.T) {
//	chainedAdapter, _ := bucket.New("testbucket", "/folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
//	mockS3 := new(MockS3Client)
//	chainedAdapter.(*adapter.AbstractAdapter).Client = mockS3
//	setInput := &s3.PutObjectInput{
//		Bucket:      aws.String("testbucket"),
//		Key:         aws.String("/folder/key.json"),
//		Body:        bytes.NewReader([]byte(`{"value":"foo"}`)),
//		ContentType: aws.String(bucket.MimeTypeJson),
//	}
//	setOutput := &s3.PutObjectOutput{}
//	mockS3.On("PutObject", context.TODO(), setInput).Return(setOutput, nil)
//
//	_, err := chainedAdapter.SetItem("key", `{"value":"foo"}`)
//	assert.NoError(t, err)
//
//	sut := memory.New("two:", time.Second*60, time.Second*120)
//	sut.(storage.Chainable).ChainAdapter(chainedAdapter)
//	//check that the parent adapter does not have keys
//	adapterClient := sut.(*adapter.AbstractAdapter).Client.(*cache.Cache)
//	_, found := adapterClient.Get("two:key1")
//	assert.False(t, found)
//	_, found = adapterClient.Get("two:key2")
//	assert.False(t, found)
//	_, found = adapterClient.Get("two:key2")
//	assert.False(t, found)
//
//	ret, err := sut.GetItems([]string{"key1", "key2", "key3"})
//	assert.NoError(t, err)
//
//	//check that the parent adapter now has keys
//	_, found = adapterClient.Get("two:key1")
//	assert.True(t, found)
//	_, found = adapterClient.Get("two:key2")
//	assert.True(t, found)
//	_, found = adapterClient.Get("two:key2")
//	assert.True(t, found)
//}

func TestS3Adapter_CheckAndSetItem_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)
	ok, err := sut.CheckAndSetItem("key", "value")
	assert.False(t, ok)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrNotImplemented)
}

func TestS3Adapter_CheckAndSetMultipleItems_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)
	ret, err := sut.CheckAndSetItems(map[string]any{"key1": "value", "key2": 2})
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrNotImplemented)
	assert.Empty(t, ret)
}

func TestS3Adapter_TouchItem_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)

	assert.False(t, sut.TouchItem("bar"))
}

func TestS3Adapter_TouchMultipleItems_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)

	keys := sut.TouchItems([]string{"key1", "key2"})
	assert.Empty(t, keys)
}

func TestS3Adapter_RemoveItem(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	assert.True(t, sut.RemoveItem("foo"))
	val, err := sut.GetItem("foo")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrKeyNotFound)
	assert.Equal(t, nil, val)

	val, err = sut.GetItem("bar")
	assert.NoError(t, err)
	assert.Equal(t, "bop", val)
}

func TestS3Adapter_RemoveMultipleItems(t *testing.T) {
	sut := memory.New("", time.Second*60, time.Second*120)
	_, err := sut.SetItems(map[string]any{
		"foo": "bar",
		"bar": "bop",
	})
	assert.NoError(t, err)

	keys := sut.RemoveItems([]string{"foo", "bar"})
	assert.True(t, slices.Contains(keys, "foo"))
	assert.True(t, slices.Contains(keys, "bar"))
}

func TestS3Adapter_IncrementValidNumber_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)

	_, err = sut.Increment("key1", int64(1))
	assert.Error(t, err)
}

func TestS3Adapter_DecrementValidNumber_NotSupported(t *testing.T) {
	sut, err := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	assert.NoError(t, err)

	_, err = sut.Decrement("key1", int64(1))
	assert.Error(t, err)
}

func TestS3Adapter_GetClient(t *testing.T) {
	sut, _ := bucket.New("testbucket", "folder/", ".json", bucket.MimeTypeJson, "eu-west-2")
	client := sut.(*adapter.AbstractAdapter).Client.(*s3.Client)
	assert.IsType(t, s3.Client{}, *client)
}
