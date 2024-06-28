package matchmaking

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

func (commonProtocol *CommonProtocol) getSessionURLs(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.RLock()
	gathering, _, participants, _, nexError := database.FindGatheringByID(commonProtocol.db, gid.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.RUnlock()
		return nil, nexError
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if !slices.Contains(participants, connection.PID().Value()) {
		common_globals.MatchmakingMutex.RUnlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	host := endpoint.FindConnectionByPID(gathering.HostPID.Value())
	if host == nil {
		// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
		common_globals.Logger.Warning("Host client not found, trying with owner client")
		host = endpoint.FindConnectionByPID(gathering.OwnerPID.Value())
		if host == nil {
			// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
			common_globals.Logger.Error("Owner client not found")
		}
	}

	common_globals.MatchmakingMutex.RUnlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	// * If no host was found, return an empty list of station URLs
	if host == nil {
		stationURLs := types.NewList[*types.StationURL]()
		stationURLs.Type = types.NewStationURL("")
		stationURLs.WriteTo(rmcResponseStream)
	} else {
		host.StationURLs.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetSessionURLs != nil {
		go commonProtocol.OnAfterGetSessionURLs(packet, gid)
	}

	return rmcResponse, nil
}
