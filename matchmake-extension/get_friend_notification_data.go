package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) getFriendNotificationData(err error, packet nex.PacketInterface, callID uint32, uiType *types.PrimitiveS32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	
	if common_globals.GetUserFriendPIDsHandler == nil {
		common_globals.Logger.Error("Missing GetUserFriendPIDsHandler!")
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	var friendList []uint32
	if len(friendList) == 0 {
		friendList = common_globals.GetUserFriendPIDsHandler(uint32(connection.PID().Value())) // TODO - This grpc method needs to support the Switch
	}

	dataList := types.NewList[*notifications_types.NotificationEvent]()
	for _, pid := range friendList {
		if notificationEvent, ok := common_globals.NotificationDatas[uint64(pid)]; ok {
			if (notificationEvent.Type.Value / 1000) == uint32(uiType.Value){
				dataList.Append(notificationEvent)
			}
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	dataList.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetFriendNotificationData
	rmcResponse.CallID = callID

	/*if commonProtocol.OnAfterJoinMatchmakeSession != nil {
		go commonProtocol.OnAfterJoinMatchmakeSession(packet, gid, strMessage)
	}*/

	return rmcResponse, nil
}
