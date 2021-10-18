// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"sync"
)

// Ensure, that S3UploaderMock does implement handler.S3Uploader.
// If this is not the case, regenerate this file with moq.
var _ handler.S3Uploader = &S3UploaderMock{}

// S3UploaderMock is a mock implementation of handler.S3Uploader.
//
// 	func TestSomethingThatUsesS3Uploader(t *testing.T) {
//
// 		// make and configure a mocked handler.S3Uploader
// 		mockedS3Uploader := &S3UploaderMock{
// 			BucketNameFunc: func() string {
// 				panic("mock out the BucketName method")
// 			},
// 			UploadFunc: func(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
// 				panic("mock out the Upload method")
// 			},
// 			UploadWithPSKFunc: func(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error) {
// 				panic("mock out the UploadWithPSK method")
// 			},
// 		}
//
// 		// use mockedS3Uploader in code that requires handler.S3Uploader
// 		// and then make assertions.
//
// 	}
type S3UploaderMock struct {
	// BucketNameFunc mocks the BucketName method.
	BucketNameFunc func() string

	// UploadFunc mocks the Upload method.
	UploadFunc func(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error)

	// UploadWithPSKFunc mocks the UploadWithPSK method.
	UploadWithPSKFunc func(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error)

	// calls tracks calls to the methods.
	calls struct {
		// BucketName holds details about calls to the BucketName method.
		BucketName []struct {
		}
		// Upload holds details about calls to the Upload method.
		Upload []struct {
			// Input is the input argument value.
			Input *s3manager.UploadInput
			// Options is the options argument value.
			Options []func(*s3manager.Uploader)
		}
		// UploadWithPSK holds details about calls to the UploadWithPSK method.
		UploadWithPSK []struct {
			// Input is the input argument value.
			Input *s3manager.UploadInput
			// Psk is the psk argument value.
			Psk []byte
		}
	}
	lockBucketName    sync.RWMutex
	lockUpload        sync.RWMutex
	lockUploadWithPSK sync.RWMutex
}

// BucketName calls BucketNameFunc.
func (mock *S3UploaderMock) BucketName() string {
	if mock.BucketNameFunc == nil {
		panic("S3UploaderMock.BucketNameFunc: method is nil but S3Uploader.BucketName was just called")
	}
	callInfo := struct {
	}{}
	mock.lockBucketName.Lock()
	mock.calls.BucketName = append(mock.calls.BucketName, callInfo)
	mock.lockBucketName.Unlock()
	return mock.BucketNameFunc()
}

// BucketNameCalls gets all the calls that were made to BucketName.
// Check the length with:
//     len(mockedS3Uploader.BucketNameCalls())
func (mock *S3UploaderMock) BucketNameCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockBucketName.RLock()
	calls = mock.calls.BucketName
	mock.lockBucketName.RUnlock()
	return calls
}

// Upload calls UploadFunc.
func (mock *S3UploaderMock) Upload(input *s3manager.UploadInput, options ...func(*s3manager.Uploader)) (*s3manager.UploadOutput, error) {
	if mock.UploadFunc == nil {
		panic("S3UploaderMock.UploadFunc: method is nil but S3Uploader.Upload was just called")
	}
	callInfo := struct {
		Input   *s3manager.UploadInput
		Options []func(*s3manager.Uploader)
	}{
		Input:   input,
		Options: options,
	}
	mock.lockUpload.Lock()
	mock.calls.Upload = append(mock.calls.Upload, callInfo)
	mock.lockUpload.Unlock()
	return mock.UploadFunc(input, options...)
}

// UploadCalls gets all the calls that were made to Upload.
// Check the length with:
//     len(mockedS3Uploader.UploadCalls())
func (mock *S3UploaderMock) UploadCalls() []struct {
	Input   *s3manager.UploadInput
	Options []func(*s3manager.Uploader)
} {
	var calls []struct {
		Input   *s3manager.UploadInput
		Options []func(*s3manager.Uploader)
	}
	mock.lockUpload.RLock()
	calls = mock.calls.Upload
	mock.lockUpload.RUnlock()
	return calls
}

// UploadWithPSK calls UploadWithPSKFunc.
func (mock *S3UploaderMock) UploadWithPSK(input *s3manager.UploadInput, psk []byte) (*s3manager.UploadOutput, error) {
	if mock.UploadWithPSKFunc == nil {
		panic("S3UploaderMock.UploadWithPSKFunc: method is nil but S3Uploader.UploadWithPSK was just called")
	}
	callInfo := struct {
		Input *s3manager.UploadInput
		Psk   []byte
	}{
		Input: input,
		Psk:   psk,
	}
	mock.lockUploadWithPSK.Lock()
	mock.calls.UploadWithPSK = append(mock.calls.UploadWithPSK, callInfo)
	mock.lockUploadWithPSK.Unlock()
	return mock.UploadWithPSKFunc(input, psk)
}

// UploadWithPSKCalls gets all the calls that were made to UploadWithPSK.
// Check the length with:
//     len(mockedS3Uploader.UploadWithPSKCalls())
func (mock *S3UploaderMock) UploadWithPSKCalls() []struct {
	Input *s3manager.UploadInput
	Psk   []byte
} {
	var calls []struct {
		Input *s3manager.UploadInput
		Psk   []byte
	}
	mock.lockUploadWithPSK.RLock()
	calls = mock.calls.UploadWithPSK
	mock.lockUploadWithPSK.RUnlock()
	return calls
}
