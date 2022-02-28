package handler_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler/mock"
	. "github.com/smartystreets/goconvey/convey"
)

type stubWriter struct {
	data []byte
}

func (w *stubWriter) Write(p []byte) (n int, err error) {
	w.data = p
	return len(w.data), nil
}

type ioReadCloserMock struct {
	returnedBytes []byte
	returnedError error
}

func (r *ioReadCloserMock) Read(p []byte) (n int, err error) {
	// p = r.returnedBytes
	n = len(r.returnedBytes)
	return n, r.returnedError
}

func (r *ioReadCloserMock) Close() error {
	return nil
}

type ioWriterMock struct {
	receivedBytes []byte
	returnedError error
}

func (w *ioWriterMock) Write(p []byte) (n int, err error) {
	w.receivedBytes = p
	n = len(w.receivedBytes)
	return n, w.returnedError
}

func TestStreamWriter_WriteContent(t *testing.T) {

	var csvCreatedEvent = &event.CantabularCsvCreated{
		InstanceID: "",
		DatasetID:  "",
		Edition:    "",
		Version:    "",
		RowCount:   1}

	ctx := context.Background()

	Convey("should return expected error if s3 Get fails for published path", t, func() {
		s3Mock := &mock.S3ClientMock{
			GetFunc: func(filename string) (io.ReadCloser, *int64, error) {
				return nil, nil, errors.New("error with S3")
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, s3Mock, nil, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, nil, true)

		So(err.Error(), ShouldContainSubstring, "error with S3")
		So(length, ShouldEqual, 0)
	})

	Convey("should return expected error if vault client read key returns and error for non published path", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "", errors.New("key not found")
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, nil, false)

		So(err.Error(), ShouldContainSubstring, "key not found")
		So(length, ShouldEqual, 0)
	})

	Convey("should return expected error if s3 client GetWithPSK returns an error for non published path", t, func() {
		s3Mock := &mock.S3ClientMock{
			GetWithPSKFunc: func(filename string, psk []byte) (io.ReadCloser, *int64, error) {
				return nil, nil, errors.New("PSK error")
			},
		}

		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "0123456789ABCDEF", nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, s3Mock, nil, vaultMock, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, nil, false)

		So(err.Error(), ShouldContainSubstring, "PSK error")
		So(length, ShouldEqual, 0)
	})

	Convey("should return expected error if s3reader returns an error for published path", t, func() {
		s3Mock := &mock.S3ClientMock{
			GetFunc: func(filename string) (io.ReadCloser, *int64, error) {
				return &ioReadCloserMock{
						[]byte{},
						errors.New("ioReadClose Read mock fail"),
					},
					nil,
					nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, s3Mock, nil, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, nil, true)

		So(err.Error(), ShouldContainSubstring, "ioReadClose Read mock fail")
		So(length, ShouldEqual, 0)
	})

	Convey("should return expected error if writer.write in io.Copy returns an error for published path", t, func() {
		s3Mock := &mock.S3ClientMock{
			GetFunc: func(filename string) (io.ReadCloser, *int64, error) {
				return ioutil.NopCloser(strings.NewReader("Test")), nil, nil
			},
		}

		w := &ioWriterMock{
			returnedError: errors.New("writer error"),
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, s3Mock, nil, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, w, true)

		So(err.Error(), ShouldContainSubstring, "writer error")
		So(length, ShouldEqual, 0)
	})

	Convey("should successfully write bytes from s3Reader to the provided writer for published path", t, func() {
		var testLen int64 = 10
		s3Mock := &mock.S3ClientMock{
			GetFunc: func(filename string) (io.ReadCloser, *int64, error) {
				return ioutil.NopCloser(strings.NewReader("1, 2, 3, 4")), &testLen, nil
			},
		}

		w := &ioWriterMock{
			returnedError: nil,
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, s3Mock, nil, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, w, true)

		So(err, ShouldBeNil)
		So(length, ShouldEqual, testLen)
		So(w.receivedBytes, ShouldResemble, []byte("1, 2, 3, 4"))
	})

	Convey("should successfully write bytes from s3Reader to the provided writer for non published path", t, func() {
		var testLen int64 = 10
		s3Mock := &mock.S3ClientMock{
			GetWithPSKFunc: func(filename string, psk []byte) (io.ReadCloser, *int64, error) {
				return ioutil.NopCloser(strings.NewReader("1, 2, 3, 4")), &testLen, nil
			},
		}

		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "0123456789ABCDEF", nil
			},
		}

		w := &ioWriterMock{
			returnedError: nil,
		}

		h := handler.NewXlsxCreate(testCfg(), nil, s3Mock, nil, vaultMock, nil)

		length, err := h.StreamAndWrite(ctx, "", csvCreatedEvent, w, false)

		So(err, ShouldBeNil)
		So(length, ShouldEqual, testLen)
		So(w.receivedBytes, ShouldResemble, []byte("1, 2, 3, 4"))
		So(s3Mock.GetWithPSKCalls(), ShouldHaveLength, 1)
		So(s3Mock.GetWithPSKCalls()[0].Psk, ShouldResemble, []byte{0x01, 0x023, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
	})
}

func Test_GetVaultKeyForFile(t *testing.T) {

	var csvCreatedEvent = &event.CantabularCsvCreated{
		InstanceID: "",
		DatasetID:  "",
		Edition:    "",
		Version:    "",
		RowCount:   1}

	Convey("should return error event is nil", t, func() {
		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, nil, nil)

		psk, err := h.GetVaultKeyForCSVFile(nil)

		So(psk, ShouldBeNil)
		So(err.Error(), ShouldEqual, ": nil event not allowed") // the ': ' is added in wrapping the stack
	})

	Convey("should return expected error if vaultClient.ReadKey is unsuccessful", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "", errors.New("key not found")
			},
		}
		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil)

		psk, err := h.GetVaultKeyForCSVFile(csvCreatedEvent)

		So(psk, ShouldBeNil)
		So(err.Error(), ShouldContainSubstring, "key not found")
	})

	Convey("should return expected error if the vault key cannot be hex decoded", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "non hex string", nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil)

		psk, err := h.GetVaultKeyForCSVFile(csvCreatedEvent)

		So(psk, ShouldBeNil)
		So(err.Error(), ShouldContainSubstring, "failed in DecodeString")
	})

	Convey("should return expected vault key for successful requests", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "0123456789ABCDEF", nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil)

		psk, err := h.GetVaultKeyForCSVFile(csvCreatedEvent)

		So(psk, ShouldResemble, []byte{0x01, 0x023, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
		So(err, ShouldBeNil)
	})

}
