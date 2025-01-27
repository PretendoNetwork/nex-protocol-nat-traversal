package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

// LogNotificationData logs the update of the notification data of a user with UpdateNotificationData
func LogNotificationData(db *sql.DB, notificationData notifications_types.NotificationEvent) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.notification_data (
		date,
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
		$5,
		$6
	)`,
		eventTime,
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
