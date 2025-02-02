package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetNotificationDatas gets the notification datas that belong to friends of the user and match with any of the given types
func GetNotificationDatas(manager *common_globals.MatchmakingManager, sourcePID types.PID, notificationTypes []uint32) ([]notifications_types.NotificationEvent, *nex.Error) {
	dataList := make([]notifications_types.NotificationEvent, 0)

	var friendList []uint32
	if manager.GetUserFriendPIDs != nil {
		friendList = manager.GetUserFriendPIDs(uint32(sourcePID))
	} else {
		common_globals.Logger.Warning("GetNotificationDatas missing manager.GetUserFriendPIDs!")
	}

	// * No friends to check
	if len(friendList) == 0 {
		return dataList, nil
	}

	rows, err := manager.Database.Query(`SELECT
		source_pid,
		type,
		param_1,
		param_2,
		param_str
		FROM matchmaking.notifications WHERE active=true AND source_pid=ANY($1) AND type=ANY($2)
	`, pqextended.Array(friendList), pqextended.Array(notificationTypes))
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		notificationData := notifications_types.NewNotificationEvent()

		err = rows.Scan(
			&notificationData.PIDSource,
			&notificationData.Type,
			&notificationData.Param1,
			&notificationData.Param2,
			&notificationData.StrParam,
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		dataList = append(dataList, notificationData)
	}

	rows.Close()

	return dataList, nil
}
