package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) createCommunity(err error, packet nex.PacketInterface, callID uint32, community match_making_types.PersistentGathering, strMessage types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if !common_globals.CheckValidPersistentGathering(community) {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	createdPersistentGatherings, nexError := database.GetCreatedPersistentGatherings(commonProtocol.manager, connection.PID())
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if createdPersistentGatherings >= commonProtocol.PersistentGatheringCreationMax {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PersistentGatheringCreationMax, "change_error")
	}

	nexError = database.CreatePersistentGathering(commonProtocol.manager, connection, &community)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	// TODO - Is this right? Mario Kart 7 sets 0 max participants
	if community.MaximumParticipants > 0 {
		_, nexError = match_making_database.JoinGathering(commonProtocol.manager, uint32(community.ID), connection, 1, string(strMessage))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	community.ID.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateCommunity
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCreateCommunity != nil {
		go commonProtocol.OnAfterCreateCommunity(packet, community, strMessage)
	}

	return rmcResponse, nil
}
