package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) autoMatchmakeWithParamPostpone(err error, packet nex.PacketInterface, callID uint32, autoMatchmakeParam *match_making_types.AutoMatchmakeParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// * Process additionalParticipants early so we can cancel matchmaking if it's wrong
	// * additionalParticipants is used by Splatoon to move Fest teams around between rooms
	oldGid := autoMatchmakeParam.GIDForParticipationCheck.Value
	oldGathering := common_globals.Sessions[oldGid]
	// * Check the caller is actually in the old session
	useAdditionalParticipants := oldGathering != nil && oldGathering.ConnectionIDs.Has(connection.ID)

	// * Check the other participants are actually in the old session, and note down the connection IDs
	additionalParticipants := make([]uint32, autoMatchmakeParam.AdditionalParticipants.Length()+1)

	// * If using additionalParticipants we'll *move* everyone, rather than disconnect/reconnect.
	// * This prevents issues from host migration and disconnected notifications.
	if useAdditionalParticipants {
		for i, pid := range autoMatchmakeParam.AdditionalParticipants.Slice() {
			targetConnection := endpoint.FindConnectionByPID(pid.Value())
			if targetConnection == nil || !oldGathering.ConnectionIDs.Has(targetConnection.ID) {
				// * This code is so early in the matchmaking process so this error can be here
				return nil, nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "Couldn't find connection")
			}

			additionalParticipants[i] = targetConnection.ID
		}

		// * Include the host too!
		additionalParticipants[len(additionalParticipants)-1] = connection.ID
	} else {
		// * A client may disconnect from a session without leaving reliably,
		// * so let's make sure the client is removed from the session.
		common_globals.RemoveConnectionFromAllSessions(connection)
	}

	matchmakeSession := autoMatchmakeParam.SourceMatchmakeSession

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(connection.PID(), autoMatchmakeParam.LstSearchCriteria.Slice(), commonProtocol.GameSpecificMatchmakeSessionSearchCriteriaChecks)
	var session *common_globals.CommonMatchmakeSession

	if len(sessions) == 0 {
		var errCode *nex.Error
		session, errCode = common_globals.CreateSessionByMatchmakeSession(matchmakeSession, nil, connection.PID())
		if errCode != nil {
			common_globals.Logger.Error(errCode.Error())
			return nil, errCode
		}
	} else {
		session = sessions[0]
	}

	if useAdditionalParticipants {
		// * Move everyone. They're still connected to their old session
		errCode := common_globals.MovePlayersToSession(session, additionalParticipants, connection, autoMatchmakeParam.JoinMessage.Value)
		if errCode != nil {
			common_globals.Logger.Error(errCode.Error())
			return nil, errCode
		}
	} else {
		errCode := common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, autoMatchmakeParam.JoinMessage.Value)
		if errCode != nil {
			common_globals.Logger.Error(errCode.Error())
			return nil, errCode
		}
	}

	matchmakeDataHolder := types.NewAnyDataHolder()

	matchmakeDataHolder.TypeName = types.NewString("MatchmakeSession")
	matchmakeDataHolder.ObjectData = session.GameMatchmakeSession.Copy()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	session.GameMatchmakeSession.WriteTo(rmcResponseStream)

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
