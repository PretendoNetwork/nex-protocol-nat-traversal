package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

func (commonProtocol *CommonProtocol) findOfficialCommunity(err error, packet nex.PacketInterface, callID uint32, isAvailableOnly types.Bool, resultRange types.ResultRange) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	communities, nexError := database.GetOfficialCommunities(commonProtocol.manager, connection.PID(), bool(isAvailableOnly), resultRange)
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
	rmcResponse.MethodID = matchmake_extension.MethodFindOfficialCommunity
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterFindOfficialCommunity != nil {
		go commonProtocol.OnAfterFindOfficialCommunity(packet, isAvailableOnly, resultRange)
	}

	return rmcResponse, nil
}
