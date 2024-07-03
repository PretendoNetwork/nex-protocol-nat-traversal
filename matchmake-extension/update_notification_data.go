package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) updateNotificationData(err error, packet nex.PacketInterface, callID uint32, uiType *types.PrimitiveU32, uiParam1 *types.PrimitiveU32, uiParam2 *types.PrimitiveU32, strParam *types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	
	event := notifications_types.NewNotificationEvent()
	event.PIDSource = connection.PID()
	event.Type = types.NewPrimitiveU32(uiType.Value * 1000)
	event.Param1 = uiParam1
	event.Param2 = uiParam2
	event.StrParam = strParam
	event.Param3 = types.NewPrimitiveU32(0xffffffff)
	
	common_globals.NotificationDatas[connection.PID().Value()] = event

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateNotificationData
	rmcResponse.CallID = callID

	/*if commonProtocol.OnAfterJoinMatchmakeSession != nil {
		go commonProtocol.OnAfterJoinMatchmakeSession(packet, gid, strMessage)
	}*/

	return rmcResponse, nil
}
