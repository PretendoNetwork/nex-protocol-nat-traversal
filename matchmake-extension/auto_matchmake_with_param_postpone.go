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

func (commonProtocol *CommonProtocol) autoMatchmakeWithParamPostpone(err error, packet nex.PacketInterface, callID uint32, autoMatchmakeParam match_making_types.AutoMatchmakeParam) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.CleanupMatchmakeSessionSearchCriterias == nil {
		common_globals.Logger.Warning("MatchmakeExtension::AutoMatchmakeWithParam_Postpone missing CleanupMatchmakeSessionSearchCriterias!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if !common_globals.CheckValidMatchmakeSession(autoMatchmakeParam.SourceMatchmakeSession) {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(autoMatchmakeParam.JoinMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	if autoMatchmakeParam.GIDForParticipationCheck != 0 {
		// * Check that all new participants are participating in the specified gathering ID
		nexError := database.CheckGatheringForParticipation(commonProtocol.manager, uint32(autoMatchmakeParam.GIDForParticipationCheck), append(autoMatchmakeParam.AdditionalParticipants, connection.PID()))
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	}

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	database.EndMatchmakeSessionsParticipation(commonProtocol.manager, connection)

	commonProtocol.CleanupMatchmakeSessionSearchCriterias(autoMatchmakeParam.LstSearchCriteria)

	resultRange := types.NewResultRange()
	resultRange.Length = 1
	resultSessions, nexError := database.FindMatchmakeSessionBySearchCriteria(commonProtocol.manager, connection, autoMatchmakeParam.LstSearchCriteria, resultRange, &autoMatchmakeParam.SourceMatchmakeSession)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	var resultSession match_making_types.MatchmakeSession
	if len(resultSessions) == 0 {
		resultSession = autoMatchmakeParam.SourceMatchmakeSession.Copy().(match_making_types.MatchmakeSession)
		nexError = database.CreateMatchmakeSession(commonProtocol.manager, connection, &resultSession)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	} else {
		resultSession = resultSessions[0]

		// TODO - What should really happen here?
		if resultSession.UserPasswordEnabled || resultSession.SystemPasswordEnabled {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
		}
	}

	participants, nexError := database.JoinMatchmakeSessionWithParticipants(commonProtocol.manager, resultSession, connection, autoMatchmakeParam.AdditionalParticipants, string(autoMatchmakeParam.JoinMessage), constants.JoinMatchmakeSessionBehaviorJoinMyself)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	resultSession.ParticipationCount = types.NewUInt32(participants)

	commonProtocol.manager.Mutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	resultSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakeWithParamPostpone
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterAutoMatchmakeWithParamPostpone != nil {
		go commonProtocol.OnAfterAutoMatchmakeWithParamPostpone(packet, autoMatchmakeParam)
	}

	return rmcResponse, nil
}
