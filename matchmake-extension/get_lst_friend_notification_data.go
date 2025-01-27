package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
)

func (commonProtocol *CommonProtocol) getlstFriendNotificationData(err error, packet nex.PacketInterface, callID uint32, lstTypes types.List[types.UInt32]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	notificationTypes := make([]uint32, len(lstTypes))
	for i, notificationType := range lstTypes {
		// * This method can only receive notifications within the range 101-108, which are reserved for game-specific notifications
		if notificationType < 101 || notificationType > 108 {
			return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
		}

		notificationTypes[i] = uint32(notificationType)
	}

	commonProtocol.manager.Mutex.RLock()

	notificationDatas, nexError := database.GetNotificationDatas(commonProtocol.manager, connection.PID(), notificationTypes)
	if nexError != nil {
		commonProtocol.manager.Mutex.RUnlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.RUnlock()

	dataList := types.NewList[notifications_types.NotificationEvent]()
	dataList = notificationDatas

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	dataList.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetlstFriendNotificationData
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateNotificationData != nil {
		go commonProtocol.OnAfterGetlstFriendNotificationData(packet, lstTypes)
	}

	return rmcResponse, nil
}
