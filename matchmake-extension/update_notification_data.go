package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/tracking"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

func (commonProtocol *CommonProtocol) updateNotificationData(err error, packet nex.PacketInterface, callID uint32, uiType types.UInt32, uiParam1 types.UInt32, uiParam2 types.UInt32, strParam types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	// * This method can only send notifications within the range 101-108, which are reserved for game-specific notifications
	if uiType < 101 || uiType > 108 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	// * All strings must have a length lower than 256
	if len(strParam) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	notificationData := notifications_types.NewNotificationEvent()
	notificationData.PIDSource = connection.PID()
	notificationData.Type = types.NewUInt32(notifications.BuildNotificationType(uint32(uiType), 0))
	notificationData.Param1 = uiParam1
	notificationData.Param2 = uiParam2
	notificationData.StrParam = strParam

	commonProtocol.manager.Mutex.Lock()

	nexError := database.UpdateNotificationData(commonProtocol.manager, notificationData)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	nexError = tracking.LogNotificationData(commonProtocol.manager.Database, notificationData)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	// * If the friends are connected, try to send the notifications directly aswell. This is observed on Mario Tennis Open
	var friendList []uint32
	if commonProtocol.manager.GetUserFriendPIDs != nil {
		friendList = commonProtocol.manager.GetUserFriendPIDs(uint32(connection.PID()))
	}

	if len(friendList) != 0 {
		var targets []uint64
		for _, pid := range friendList {
			targets = append(targets, uint64(pid))
		}

		common_globals.SendNotificationEvent(endpoint, notificationData, targets)
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateNotificationData
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateNotificationData != nil {
		go commonProtocol.OnAfterUpdateNotificationData(packet, uiType, uiParam1, uiParam2, strParam)
	}

	return rmcResponse, nil
}
