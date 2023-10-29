package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func searchObject(err error, client *nex.Client, callID uint32, param *datastore_types.DataStoreSearchParam) uint32 {
	if commonDataStoreProtocol.getObjectInfosByDataStoreSearchParamHandler == nil {
		common_globals.Logger.Warning("GetObjectInfosByDataStoreSearchParam not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.verifyObjectPermissionHandler == nil {
		common_globals.Logger.Warning("VerifyObjectPermission not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	// * This is likely game-specific. Also developer note:
	// * Please keep in mind that no results is allowed. errCode
	// * should NEVER be DataStore::NotFound!
	// *
	// * DataStoreSearchParam contains a ResultRange to limit the
	// * returned results. TotalCount is the total matching objects
	// * in the database, whereas objects is the limited results
	objects, totalCount, errCode := commonDataStoreProtocol.getObjectInfosByDataStoreSearchParamHandler(param)
	if errCode != 0 {
		return errCode
	}

	pSearchResult := datastore_types.NewDataStoreSearchResult()

	pSearchResult.TotalCount = totalCount
	pSearchResult.Result = make([]*datastore_types.DataStoreMetaInfo, 0, len(objects))
	pSearchResult.TotalCountType = uint8(param.DataType) // TODO - Is this right?

	for _, object := range objects {
		errCode = commonDataStoreProtocol.verifyObjectPermissionHandler(object.OwnerID, client.PID(), object.Permission)
		if errCode != 0 {
			continue
		}

		// * This is kind of backwards.
		// * The database pulls this data
		// * by default, so it can be done
		// * in a single query. So instead
		// * of checking if a flag *IS*
		// * set, and conditionally *ADDING*
		// * the fields, we check if a flag
		// * is *NOT* set and conditionally
		// * *REMOVE* the field
		if param.ResultOption&0x1 == 0 {
			object.Tags = make([]string, 0)
		}

		if param.ResultOption&0x2 == 0 {
			object.Ratings = make([]*datastore_types.DataStoreRatingInfoWithSlot, 0)
		}

		if param.ResultOption&0x4 == 0 {
			object.MetaBinary = make([]byte, 0)
		}

		pSearchResult.Result = append(pSearchResult.Result, object)
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	rmcResponseStream.WriteStructure(pSearchResult)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodSearchObject, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
