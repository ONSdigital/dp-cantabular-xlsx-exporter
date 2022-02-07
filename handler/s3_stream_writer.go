package handler

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/event"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/pkg/errors"
)

// StreamAndWrite decrypt and stream the request file writing the content to the provided io.Writer.
func (h *XlsxCreate) StreamAndWrite(ctx context.Context, filenameCsv string, event *event.CantabularCsvCreated, w io.Writer, isPublished bool) (length int64, err error) {
	var s3ReadCloser io.ReadCloser
	var lengthPtr *int64

	if isPublished {
		s3ReadCloser, lengthPtr, err = h.s3Public.Get(filenameCsv)
		if err != nil {
			return 0, errors.Wrap(err, "failed in Published Get")
		}
	} else {
		psk, err := h.GetVaultKeyForCSVFile(event)
		if err != nil {
			return 0, errors.Wrap(err, "failed in getVaultKeyForCSVFile")
		}

		s3ReadCloser, lengthPtr, err = h.s3Private.GetWithPSK(filenameCsv, psk)
		if err != nil {
			return 0, errors.Wrap(err, "failed in GetWithPSK")
		}
	}

	if lengthPtr != nil {
		length = *lengthPtr
	}

	defer closeAndLogError(ctx, s3ReadCloser)

	_, err = io.Copy(w, s3ReadCloser)
	if err != nil {
		return 0, errors.Wrap(err, "failed in io.Copy")
	}

	return length, nil
}

func closeAndLogError(ctx context.Context, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Error(ctx, "error closing io.Closer", err)
	}
}

func (h *XlsxCreate) GetVaultKeyForCSVFile(event *event.CantabularCsvCreated) ([]byte, error) {
	if event == nil {
		return nil, ErrorStack("nil event not allowed")
	}
	vaultPath := fmt.Sprintf("%s/%s-%s-%s.csv", h.cfg.VaultPath, event.DatasetID, event.Edition, event.Version)

	pskStr, err := h.vaultClient.ReadKey(vaultPath, "key")
	if err != nil {
		return nil, errors.Wrapf(err, "for 'vaultPath': %s, failed in ReadKey", vaultPath)
	}

	psk, err := hex.DecodeString(pskStr)
	if err != nil {
		return nil, errors.Wrap(err, "failed in DecodeString")
	}

	return psk, nil
}
