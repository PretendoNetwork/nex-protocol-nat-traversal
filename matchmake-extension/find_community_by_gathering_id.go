package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) findCommunityByGatheringID(err error, packet nex.PacketInterface, callID uint32, lstGID types.List[types.UInt32]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	var gatheringIDs []uint32
	for _, gatheringID := range lstGID {
		gatheringIDs = append(gatheringIDs, uint32(gatheringID))
	}

	communities, nexError := database.GetPersistentGatheringsByID(commonProtocol.manager, connection.PID(), gatheringIDs)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	lstCommunity := types.NewList[match_making_types.PersistentGathering]()

	for _, community := range communities {
		// * Scrap persistent gathering password
		community.Password = ""
	}

	lstCommunity = communities

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstCommunity.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodFindCommunityByGatheringID
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterFindCommunityByGatheringID != nil {
		go commonProtocol.OnAfterFindCommunityByGatheringID(packet, lstGID)
	}

	return rmcResponse, nil
}
