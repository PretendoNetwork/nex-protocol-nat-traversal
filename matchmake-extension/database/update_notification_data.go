package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateNotificationData updates the notification data of the specified user and type
func UpdateNotificationData(manager *common_globals.MatchmakingManager, notificationData notifications_types.NotificationEvent) *nex.Error {
	_, err := manager.Database.Exec(`INSERT INTO matchmaking.notifications AS n (
		source_pid,
		type,
		param_1,
		param_2,
		param_str
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	) ON CONFLICT (source_pid, type) DO UPDATE SET
	param_1=$3, param_2=$4, param_str=$5, active=true WHERE n.source_pid=$1 AND n.type=$2`,
		notificationData.PIDSource,
		notificationData.Type,
		notificationData.Param1,
		notificationData.Param2,
		notificationData.StrParam,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
