package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) createMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, createMatchmakeSessionParam match_making_types.CreateMatchmakeSessionParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if !common_globals.CheckValidMatchmakeSession(createMatchmakeSessionParam.SourceMatchmakeSession) {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(createMatchmakeSessionParam.JoinMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	if createMatchmakeSessionParam.GIDForParticipationCheck != 0 {
		// * Check that all new participants are participating in the specified gathering ID
		nexError := database.CheckGatheringForParticipation(commonProtocol.manager, uint32(createMatchmakeSessionParam.GIDForParticipationCheck), append(createMatchmakeSessionParam.AdditionalParticipants, connection.PID()))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	}

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	database.EndMatchmakeSessionsParticipation(commonProtocol.manager, connection)

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(match_making_types.MatchmakeSession)
	nexError := database.CreateMatchmakeSession(commonProtocol.manager, connection, &joinedMatchmakeSession)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	participants, nexError := database.JoinMatchmakeSessionWithParticipants(commonProtocol.manager, joinedMatchmakeSession, connection, createMatchmakeSessionParam.AdditionalParticipants, string(createMatchmakeSessionParam.JoinMessage), constants.JoinMatchmakeSessionBehaviorJoinMyself)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	joinedMatchmakeSession.ParticipationCount = types.NewUInt32(participants)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	joinedMatchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCreateMatchmakeSessionWithParam != nil {
		go commonProtocol.OnAfterCreateMatchmakeSessionWithParam(packet, createMatchmakeSessionParam)
	}

	return rmcResponse, nil
}
