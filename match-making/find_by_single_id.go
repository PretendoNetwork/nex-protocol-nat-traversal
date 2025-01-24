package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

func (commonProtocol *CommonProtocol) findBySingleID(err error, packet nex.PacketInterface, callID uint32, id types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.RLock()

	gathering, _, nexError := commonProtocol.manager.GetDetailedGatheringByID(commonProtocol.manager, uint64(connection.PID()), uint32(id))
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	bResult := types.NewBool(true)
	pGathering := match_making_types.NewGatheringHolder()
	pGathering.Object = gathering.Copy().(match_making_types.GatheringInterface)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	bResult.WriteTo(rmcResponseStream)
	pGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodFindBySingleID
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterFindBySingleID != nil {
		go commonProtocol.OnAfterFindBySingleID(packet, id)
	}

	return rmcResponse, nil
}
