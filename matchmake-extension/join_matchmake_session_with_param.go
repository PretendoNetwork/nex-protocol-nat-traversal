package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) joinMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, joinMatchmakeSessionParam match_making_types.JoinMatchmakeSessionParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(joinMatchmakeSessionParam.JoinMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if joinMatchmakeSessionParam.GIDForParticipationCheck != 0 {
		// * Check that all new participants are participating in the specified gathering ID
		nexError := database.CheckGatheringForParticipation(commonProtocol.manager, uint32(joinMatchmakeSessionParam.GIDForParticipationCheck), append(joinMatchmakeSessionParam.AdditionalParticipants, connection.PID()))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	}

	joinedMatchmakeSession, systemPassword, nexError := database.GetMatchmakeSessionByID(commonProtocol.manager, endpoint, uint32(joinMatchmakeSessionParam.GID))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	// TODO - Are these the correct error codes?
	if bool(joinedMatchmakeSession.UserPasswordEnabled) && !joinMatchmakeSessionParam.StrUserPassword.Equals(joinedMatchmakeSession.UserPassword) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "change_error")
	}

	if bool(joinedMatchmakeSession.SystemPasswordEnabled) && string(joinMatchmakeSessionParam.StrSystemPassword) != systemPassword {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "change_error")
	}

	// * Allow game servers to do their own permissions checks
	if commonProtocol.CanJoinMatchmakeSession != nil {
		nexError = commonProtocol.CanJoinMatchmakeSession(commonProtocol.manager, connection.PID(), joinedMatchmakeSession)
	} else {
		nexError = common_globals.CanJoinMatchmakeSession(commonProtocol.manager, connection.PID(), joinedMatchmakeSession)
	}
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	_, nexError = database.JoinMatchmakeSessionWithParticipants(commonProtocol.manager, joinedMatchmakeSession, connection, joinMatchmakeSessionParam.AdditionalParticipants, string(joinMatchmakeSessionParam.JoinMessage), constants.JoinMatchmakeSessionBehavior(joinMatchmakeSessionParam.JoinMatchmakeSessionBehavior))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	joinedMatchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterJoinMatchmakeSessionWithParam != nil {
		go commonProtocol.OnAfterJoinMatchmakeSessionWithParam(packet, joinMatchmakeSessionParam)
	}

	return rmcResponse, nil
}
