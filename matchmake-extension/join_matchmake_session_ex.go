package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) joinMatchmakeSessionEx(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32, strMessage types.String, dontCareMyBlockList types.Bool, participationCount types.UInt16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(strMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	joinedMatchmakeSession, _, nexError := database.GetMatchmakeSessionByID(commonProtocol.manager, endpoint, uint32(gid))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	// TODO - Is this the correct error code?
	if joinedMatchmakeSession.UserPasswordEnabled || joinedMatchmakeSession.SystemPasswordEnabled {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
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

	_, nexError = database.JoinMatchmakeSession(commonProtocol.manager, joinedMatchmakeSession, connection, uint16(participationCount), string(strMessage))
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	if server.LibraryVersions.MatchMaking.GreaterOrEqual("3.0.0") {
		joinedMatchmakeSession.SessionKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionEx
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterJoinMatchmakeSessionEx != nil {
		go commonProtocol.OnAfterJoinMatchmakeSessionEx(packet, gid, strMessage, dontCareMyBlockList, participationCount)
	}

	return rmcResponse, nil
}
