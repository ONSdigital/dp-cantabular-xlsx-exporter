// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"context"
	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-cantabular-xlsx-exporter/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"sync"
)

// Ensure, that DatasetAPIClientMock does implement service.DatasetAPIClient.
// If this is not the case, regenerate this file with moq.
var _ service.DatasetAPIClient = &DatasetAPIClientMock{}

// DatasetAPIClientMock is a mock implementation of service.DatasetAPIClient.
//
// 	func TestSomethingThatUsesDatasetAPIClient(t *testing.T) {
//
// 		// make and configure a mocked service.DatasetAPIClient
// 		mockedDatasetAPIClient := &DatasetAPIClientMock{
// 			CheckerFunc: func(contextMoqParam context.Context, checkState *healthcheck.CheckState) error {
// 				panic("mock out the Checker method")
// 			},
// 			GetInstanceFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
// 				panic("mock out the GetInstance method")
// 			},
// 			GetVersionMetadataSelectionFunc: func(contextMoqParam context.Context, getVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error) {
// 				panic("mock out the GetVersionMetadataSelection method")
// 			},
// 			GetVersionsFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, downloadServiceAuthToken string, collectionID string, datasetID string, edition string, q *dataset.QueryParams) (dataset.VersionsList, error) {
// 				panic("mock out the GetVersions method")
// 			},
// 			PutVersionFunc: func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, datasetID string, edition string, version string, m dataset.Version) error {
// 				panic("mock out the PutVersion method")
// 			},
// 		}
//
// 		// use mockedDatasetAPIClient in code that requires service.DatasetAPIClient
// 		// and then make assertions.
//
// 	}
type DatasetAPIClientMock struct {
	// CheckerFunc mocks the Checker method.
	CheckerFunc func(contextMoqParam context.Context, checkState *healthcheck.CheckState) error

	// GetInstanceFunc mocks the GetInstance method.
	GetInstanceFunc func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error)

	// GetVersionMetadataSelectionFunc mocks the GetVersionMetadataSelection method.
	GetVersionMetadataSelectionFunc func(contextMoqParam context.Context, getVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error)

	// GetVersionsFunc mocks the GetVersions method.
	GetVersionsFunc func(ctx context.Context, userAuthToken string, serviceAuthToken string, downloadServiceAuthToken string, collectionID string, datasetID string, edition string, q *dataset.QueryParams) (dataset.VersionsList, error)

	// PutVersionFunc mocks the PutVersion method.
	PutVersionFunc func(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, datasetID string, edition string, version string, m dataset.Version) error

	// calls tracks calls to the methods.
	calls struct {
		// Checker holds details about calls to the Checker method.
		Checker []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// CheckState is the checkState argument value.
			CheckState *healthcheck.CheckState
		}
		// GetInstance holds details about calls to the GetInstance method.
		GetInstance []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// UserAuthToken is the userAuthToken argument value.
			UserAuthToken string
			// ServiceAuthToken is the serviceAuthToken argument value.
			ServiceAuthToken string
			// CollectionID is the collectionID argument value.
			CollectionID string
			// InstanceID is the instanceID argument value.
			InstanceID string
			// IfMatch is the ifMatch argument value.
			IfMatch string
		}
		// GetVersionMetadataSelection holds details about calls to the GetVersionMetadataSelection method.
		GetVersionMetadataSelection []struct {
			// ContextMoqParam is the contextMoqParam argument value.
			ContextMoqParam context.Context
			// GetVersionMetadataSelectionInput is the getVersionMetadataSelectionInput argument value.
			GetVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput
		}
		// GetVersions holds details about calls to the GetVersions method.
		GetVersions []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// UserAuthToken is the userAuthToken argument value.
			UserAuthToken string
			// ServiceAuthToken is the serviceAuthToken argument value.
			ServiceAuthToken string
			// DownloadServiceAuthToken is the downloadServiceAuthToken argument value.
			DownloadServiceAuthToken string
			// CollectionID is the collectionID argument value.
			CollectionID string
			// DatasetID is the datasetID argument value.
			DatasetID string
			// Edition is the edition argument value.
			Edition string
			// Q is the q argument value.
			Q *dataset.QueryParams
		}
		// PutVersion holds details about calls to the PutVersion method.
		PutVersion []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// UserAuthToken is the userAuthToken argument value.
			UserAuthToken string
			// ServiceAuthToken is the serviceAuthToken argument value.
			ServiceAuthToken string
			// CollectionID is the collectionID argument value.
			CollectionID string
			// DatasetID is the datasetID argument value.
			DatasetID string
			// Edition is the edition argument value.
			Edition string
			// Version is the version argument value.
			Version string
			// M is the m argument value.
			M dataset.Version
		}
	}
	lockChecker                     sync.RWMutex
	lockGetInstance                 sync.RWMutex
	lockGetVersionMetadataSelection sync.RWMutex
	lockGetVersions                 sync.RWMutex
	lockPutVersion                  sync.RWMutex
}

// Checker calls CheckerFunc.
func (mock *DatasetAPIClientMock) Checker(contextMoqParam context.Context, checkState *healthcheck.CheckState) error {
	if mock.CheckerFunc == nil {
		panic("DatasetAPIClientMock.CheckerFunc: method is nil but DatasetAPIClient.Checker was just called")
	}
	callInfo := struct {
		ContextMoqParam context.Context
		CheckState      *healthcheck.CheckState
	}{
		ContextMoqParam: contextMoqParam,
		CheckState:      checkState,
	}
	mock.lockChecker.Lock()
	mock.calls.Checker = append(mock.calls.Checker, callInfo)
	mock.lockChecker.Unlock()
	return mock.CheckerFunc(contextMoqParam, checkState)
}

// CheckerCalls gets all the calls that were made to Checker.
// Check the length with:
//     len(mockedDatasetAPIClient.CheckerCalls())
func (mock *DatasetAPIClientMock) CheckerCalls() []struct {
	ContextMoqParam context.Context
	CheckState      *healthcheck.CheckState
} {
	var calls []struct {
		ContextMoqParam context.Context
		CheckState      *healthcheck.CheckState
	}
	mock.lockChecker.RLock()
	calls = mock.calls.Checker
	mock.lockChecker.RUnlock()
	return calls
}

// GetInstance calls GetInstanceFunc.
func (mock *DatasetAPIClientMock) GetInstance(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, instanceID string, ifMatch string) (dataset.Instance, string, error) {
	if mock.GetInstanceFunc == nil {
		panic("DatasetAPIClientMock.GetInstanceFunc: method is nil but DatasetAPIClient.GetInstance was just called")
	}
	callInfo := struct {
		Ctx              context.Context
		UserAuthToken    string
		ServiceAuthToken string
		CollectionID     string
		InstanceID       string
		IfMatch          string
	}{
		Ctx:              ctx,
		UserAuthToken:    userAuthToken,
		ServiceAuthToken: serviceAuthToken,
		CollectionID:     collectionID,
		InstanceID:       instanceID,
		IfMatch:          ifMatch,
	}
	mock.lockGetInstance.Lock()
	mock.calls.GetInstance = append(mock.calls.GetInstance, callInfo)
	mock.lockGetInstance.Unlock()
	return mock.GetInstanceFunc(ctx, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch)
}

// GetInstanceCalls gets all the calls that were made to GetInstance.
// Check the length with:
//     len(mockedDatasetAPIClient.GetInstanceCalls())
func (mock *DatasetAPIClientMock) GetInstanceCalls() []struct {
	Ctx              context.Context
	UserAuthToken    string
	ServiceAuthToken string
	CollectionID     string
	InstanceID       string
	IfMatch          string
} {
	var calls []struct {
		Ctx              context.Context
		UserAuthToken    string
		ServiceAuthToken string
		CollectionID     string
		InstanceID       string
		IfMatch          string
	}
	mock.lockGetInstance.RLock()
	calls = mock.calls.GetInstance
	mock.lockGetInstance.RUnlock()
	return calls
}

// GetVersionMetadataSelection calls GetVersionMetadataSelectionFunc.
func (mock *DatasetAPIClientMock) GetVersionMetadataSelection(contextMoqParam context.Context, getVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput) (*dataset.Metadata, error) {
	if mock.GetVersionMetadataSelectionFunc == nil {
		panic("DatasetAPIClientMock.GetVersionMetadataSelectionFunc: method is nil but DatasetAPIClient.GetVersionMetadataSelection was just called")
	}
	callInfo := struct {
		ContextMoqParam                  context.Context
		GetVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput
	}{
		ContextMoqParam:                  contextMoqParam,
		GetVersionMetadataSelectionInput: getVersionMetadataSelectionInput,
	}
	mock.lockGetVersionMetadataSelection.Lock()
	mock.calls.GetVersionMetadataSelection = append(mock.calls.GetVersionMetadataSelection, callInfo)
	mock.lockGetVersionMetadataSelection.Unlock()
	return mock.GetVersionMetadataSelectionFunc(contextMoqParam, getVersionMetadataSelectionInput)
}

// GetVersionMetadataSelectionCalls gets all the calls that were made to GetVersionMetadataSelection.
// Check the length with:
//     len(mockedDatasetAPIClient.GetVersionMetadataSelectionCalls())
func (mock *DatasetAPIClientMock) GetVersionMetadataSelectionCalls() []struct {
	ContextMoqParam                  context.Context
	GetVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput
} {
	var calls []struct {
		ContextMoqParam                  context.Context
		GetVersionMetadataSelectionInput dataset.GetVersionMetadataSelectionInput
	}
	mock.lockGetVersionMetadataSelection.RLock()
	calls = mock.calls.GetVersionMetadataSelection
	mock.lockGetVersionMetadataSelection.RUnlock()
	return calls
}

// GetVersions calls GetVersionsFunc.
func (mock *DatasetAPIClientMock) GetVersions(ctx context.Context, userAuthToken string, serviceAuthToken string, downloadServiceAuthToken string, collectionID string, datasetID string, edition string, q *dataset.QueryParams) (dataset.VersionsList, error) {
	if mock.GetVersionsFunc == nil {
		panic("DatasetAPIClientMock.GetVersionsFunc: method is nil but DatasetAPIClient.GetVersions was just called")
	}
	callInfo := struct {
		Ctx                      context.Context
		UserAuthToken            string
		ServiceAuthToken         string
		DownloadServiceAuthToken string
		CollectionID             string
		DatasetID                string
		Edition                  string
		Q                        *dataset.QueryParams
	}{
		Ctx:                      ctx,
		UserAuthToken:            userAuthToken,
		ServiceAuthToken:         serviceAuthToken,
		DownloadServiceAuthToken: downloadServiceAuthToken,
		CollectionID:             collectionID,
		DatasetID:                datasetID,
		Edition:                  edition,
		Q:                        q,
	}
	mock.lockGetVersions.Lock()
	mock.calls.GetVersions = append(mock.calls.GetVersions, callInfo)
	mock.lockGetVersions.Unlock()
	return mock.GetVersionsFunc(ctx, userAuthToken, serviceAuthToken, downloadServiceAuthToken, collectionID, datasetID, edition, q)
}

// GetVersionsCalls gets all the calls that were made to GetVersions.
// Check the length with:
//     len(mockedDatasetAPIClient.GetVersionsCalls())
func (mock *DatasetAPIClientMock) GetVersionsCalls() []struct {
	Ctx                      context.Context
	UserAuthToken            string
	ServiceAuthToken         string
	DownloadServiceAuthToken string
	CollectionID             string
	DatasetID                string
	Edition                  string
	Q                        *dataset.QueryParams
} {
	var calls []struct {
		Ctx                      context.Context
		UserAuthToken            string
		ServiceAuthToken         string
		DownloadServiceAuthToken string
		CollectionID             string
		DatasetID                string
		Edition                  string
		Q                        *dataset.QueryParams
	}
	mock.lockGetVersions.RLock()
	calls = mock.calls.GetVersions
	mock.lockGetVersions.RUnlock()
	return calls
}

// PutVersion calls PutVersionFunc.
func (mock *DatasetAPIClientMock) PutVersion(ctx context.Context, userAuthToken string, serviceAuthToken string, collectionID string, datasetID string, edition string, version string, m dataset.Version) error {
	if mock.PutVersionFunc == nil {
		panic("DatasetAPIClientMock.PutVersionFunc: method is nil but DatasetAPIClient.PutVersion was just called")
	}
	callInfo := struct {
		Ctx              context.Context
		UserAuthToken    string
		ServiceAuthToken string
		CollectionID     string
		DatasetID        string
		Edition          string
		Version          string
		M                dataset.Version
	}{
		Ctx:              ctx,
		UserAuthToken:    userAuthToken,
		ServiceAuthToken: serviceAuthToken,
		CollectionID:     collectionID,
		DatasetID:        datasetID,
		Edition:          edition,
		Version:          version,
		M:                m,
	}
	mock.lockPutVersion.Lock()
	mock.calls.PutVersion = append(mock.calls.PutVersion, callInfo)
	mock.lockPutVersion.Unlock()
	return mock.PutVersionFunc(ctx, userAuthToken, serviceAuthToken, collectionID, datasetID, edition, version, m)
}

// PutVersionCalls gets all the calls that were made to PutVersion.
// Check the length with:
//     len(mockedDatasetAPIClient.PutVersionCalls())
func (mock *DatasetAPIClientMock) PutVersionCalls() []struct {
	Ctx              context.Context
	UserAuthToken    string
	ServiceAuthToken string
	CollectionID     string
	DatasetID        string
	Edition          string
	Version          string
	M                dataset.Version
} {
	var calls []struct {
		Ctx              context.Context
		UserAuthToken    string
		ServiceAuthToken string
		CollectionID     string
		DatasetID        string
		Edition          string
		Version          string
		M                dataset.Version
	}
	mock.lockPutVersion.RLock()
	calls = mock.calls.PutVersion
	mock.lockPutVersion.RUnlock()
	return calls
}
