package handler_test

import (
	"errors"
	"testing"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/handler/mock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	//	testVaultPath    = "secrets"
	testVaultKeyPath = "wibblevault/1234567890.csv"
	testS3Path       = "wibble/1234567890.csv"
	testErr          = errors.New("bork")
)

type StubWriter struct {
	data []byte
}

func (w *StubWriter) Write(p []byte) (n int, err error) {
	w.data = p
	return len(w.data), nil
}

func TestStreamWriter_WriteContent(t *testing.T) {
	/*	ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		var event = &event.CantabularCsvCreated{"", "", "", "", 1}

		Convey("should return expected error if s3 and vault paths are empty", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli := vaultClientNeverInvoked(ctrl)

			s := &S3StreamWriter{VaultCli: vaultCli}

			err := s.StreamAndWrite(nil, "", "", w)

			So(errors.Is(err, VaultFilenameEmptyErr), ShouldBeTrue)
		})*/

	/*	Convey("should return expected error if vault client read key returns and error", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli, expectedErr := vaultClientErrorOnReadKey(ctrl)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(errors.Is(err, expectedErr), ShouldBeTrue)
		})

		Convey("should return expected error if vault client read key returns non hex value", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli := vaultClientReturningInvalidHexString(ctrl)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(err, ShouldNotBeNil)
		})

		Convey("should return expected error if s3 client GetWithPSK returns an error", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
			s3Cli, expectedErr := s3ClientGetWithPSKReturnsError(ctrl, testS3Path, expectedPSK)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
				S3Client:  s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(errors.Is(err, expectedErr), ShouldBeTrue)
		})

		Convey("should return expected error if s3 client Get returns an error when encryption disabled", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli := vaultClientNeverInvoked(ctrl)
			s3Cli, expectedErr := s3ClientGetReturnsError(ctrl, testS3Path)

			s := &S3StreamWriter{
				VaultPath:          testVaultPath,
				VaultCli:           vaultCli,
				S3Client:           s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(errors.Is(err, expectedErr), ShouldBeTrue)
		})

		Convey("should return expected error if s3reader returns an error", t, func() {
			w := writerNeverInvoked(ctrl)
			vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
			s3ReadCloser, expectedErr := s3ReadCloserErroringOnRead(ctrl)
			s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testS3Path, expectedPSK, s3ReadCloser)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
				S3Client:  s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(errors.Is(err, expectedErr), ShouldBeTrue)
		})

		Convey("should return expected error if writer.write returns an error", t, func() {
			w, expectedErr := writerReturningErrorOnWrite(ctrl)
			vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
			s3ReadCloser := ioutil.NopCloser(strings.NewReader("1, 2, 3, 4"))
			s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testS3Path, expectedPSK, s3ReadCloser)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
				S3Client:  s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, w)

			So(errors.Is(err, expectedErr), ShouldBeTrue)
		})

		Convey("should successfully write bytes from s3Reader to the provided writer", t, func() {
			readCloser := ioutil.NopCloser(strings.NewReader("1, 2, 3, 4"))
			writer := &StubWriter{}
			vaultCli, expectedPSK := vaultClientAndValidKey(ctrl)
			s3Cli := s3ClientGetWithPSKReturnsReader(ctrl, testS3Path, expectedPSK, readCloser)

			s := &S3StreamWriter{
				VaultPath: testVaultPath,
				VaultCli:  vaultCli,
				S3Client:  s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, writer)

			So(err, ShouldBeNil)
			So(writer.data, ShouldResemble, []byte("1, 2, 3, 4"))
		})

		Convey("should successfully write bytes from s3Reader to the provided writer when encryption disabled", t, func() {
			readCloser := ioutil.NopCloser(strings.NewReader("1, 2, 3, 4"))
			writer := &StubWriter{}
			vaultCli := vaultClientNeverInvoked(ctrl)
			s3Cli := s3ClientGetReturnsReader(ctrl, testS3Path, readCloser)

			s := &S3StreamWriter{
				VaultPath:          testVaultPath,
				VaultCli:           vaultCli,
				S3Client:           s3Cli,
			}

			err := s.StreamAndWrite(nil, testS3Path, testVaultKeyPath, writer)

			So(err, ShouldBeNil)
			So(writer.data, ShouldResemble, []byte("1, 2, 3, 4"))
		})
	*/
}

func Test_GetVaultKeyForFile(t *testing.T) {

	var event = &event.CantabularCsvCreated{"", "", "", "", 1}

	Convey("should return error event is nil", t, func() {
		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, nil, nil, nil)

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
		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil, nil)

		psk, err := h.GetVaultKeyForCSVFile(event)

		So(psk, ShouldBeNil)
		So(err.Error(), ShouldContainSubstring, "key not found")
	})

	Convey("should return expected error if the vault key cannot be hex decoded", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "non hex string", nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil, nil)

		psk, err := h.GetVaultKeyForCSVFile(event)

		So(psk, ShouldBeNil)
		So(err.Error(), ShouldContainSubstring, "failed in DecodeString")
	})

	Convey("should return expected vault key for successful requests", t, func() {
		vaultMock := &mock.VaultClientMock{
			ReadKeyFunc: func(path, key string) (string, error) {
				return "0123456789ABCDEF", nil
			},
		}

		h := handler.NewXlsxCreate(testCfg(), nil, nil, nil, vaultMock, nil, nil)

		psk, err := h.GetVaultKeyForCSVFile(event)

		So(psk, ShouldResemble, []byte{0x01, 0x023, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF})
		So(err, ShouldBeNil)
	})

}

/*func vaultClientNeverInvoked(ctrl *gomock.Controller) *mocks.MockVaultClient {
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(gomock.Any(), gomock.Any()).Times(0)
	return vaultCli
}*/

/*func vaultClientErrorOnReadKey(ctrl *gomock.Controller) (*mocks.MockVaultClient, error) {
	vp := filepath.Join(testVaultPath, testVaultKeyPath)
	vaultCli := mocks.NewMockVaultClient(ctrl)
	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return("", testErr).Times(1)
	return vaultCli, testErr
}*/

/*func vaultClientAndValidKey(ctrl *gomock.Controller) (*mocks.MockVaultClient, []byte) {
	vp := filepath.Join(testVaultPath, testVaultKeyPath)
	vaultCli := mocks.NewMockVaultClient(ctrl)

	key := "one two three four"
	keyBytes := []byte(key)
	keyHexStr := hex.EncodeToString(keyBytes)

	vaultCli.EXPECT().ReadKey(vp, vaultKey).Return(keyHexStr, nil).Times(1)
	return vaultCli, keyBytes
}*/

/*func s3ClientGetWithPSKReturnsError(ctrl *gomock.Controller, key string, psk []byte) (*mocks.MockS3Client, error) {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().GetWithPSK(key, psk).Times(1).Return(nil, nil, testErr)
	return cli, testErr
}*/

/*func s3ClientGetReturnsError(ctrl *gomock.Controller, key string) (*mocks.MockS3Client, error) {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().Get(key).Times(1).Return(nil, nil, testErr)
	return cli, testErr
}

func s3ClientGetWithPSKReturnsReader(ctrl *gomock.Controller, key string, psk []byte, r S3ReadCloser) *mocks.MockS3Client {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().GetWithPSK(key, psk).Times(1).Return(r, nil, nil)
	return cli
}

func s3ClientGetReturnsReader(ctrl *gomock.Controller, key string, r S3ReadCloser) *mocks.MockS3Client {
	cli := mocks.NewMockS3Client(ctrl)
	cli.EXPECT().Get(key).Times(1).Return(r, nil, nil)
	return cli
}

func s3ReadCloserErroringOnRead(ctrl *gomock.Controller) (*mocks.MockS3ReadCloser, error) {
	r := mocks.NewMockS3ReadCloser(ctrl)
	r.EXPECT().Read(gomock.Any()).MinTimes(1).Return(0, testErr)
	r.EXPECT().Close().Times(1)
	return r, testErr
}

func writerNeverInvoked(ctrl *gomock.Controller) *mocks.MockWriter {
	w := mocks.NewMockWriter(ctrl)
	w.EXPECT().Write(gomock.Any()).Times(0).Return(0, nil)
	return w
}

func writerReturningErrorOnWrite(ctrl *gomock.Controller) (*mocks.MockWriter, error) {
	w := mocks.NewMockWriter(ctrl)
	w.EXPECT().Write(gomock.Any()).Times(1).Return(0, testErr)
	return w, testErr
}
*/
