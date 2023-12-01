package datastore

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
)

func completePostObjects(err error, packet nex.PacketInterface, callID uint32, dataIDs []uint64) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.minIOClient == nil {
		common_globals.Logger.Warning("MinIOClient not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.GetObjectSizeByDataID == nil {
		common_globals.Logger.Warning("GetObjectSizeByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.UpdateObjectUploadCompletedByDataID == nil {
		common_globals.Logger.Warning("UpdateObjectUploadCompletedByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	for _, dataID := range dataIDs {
		bucket := commonDataStoreProtocol.S3Bucket
		key := fmt.Sprintf("%s/%d.bin", commonDataStoreProtocol.s3DataKeyBase, dataID)

		objectSizeS3, err := commonDataStoreProtocol.S3ObjectSize(bucket, key)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nil, nex.Errors.DataStore.NotFound
		}

		objectSizeDB, errCode := commonDataStoreProtocol.GetObjectSizeByDataID(dataID)
		if errCode != 0 {
			return nil, errCode
		}

		if objectSizeS3 != uint64(objectSizeDB) {
			common_globals.Logger.Errorf("Object with DataID %d did not upload correctly! Mismatched sizes", dataID)
			// TODO - Is this a good error?
			return nil, nex.Errors.DataStore.Unknown
		}

		errCode = commonDataStoreProtocol.UpdateObjectUploadCompletedByDataID(dataID, true)
		if errCode != 0 {
			return nil, errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObjects
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
