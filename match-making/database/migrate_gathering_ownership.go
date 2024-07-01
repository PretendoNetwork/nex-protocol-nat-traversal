package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

// MigrateGatheringOwnership switches the owner of the gathering with a different one
func MigrateGatheringOwnership(db *sql.DB, connection *nex.PRUDPConnection, gathering *match_making_types.Gathering, participants []uint64) *nex.Error {
	var nexError *nex.Error
	var uniqueParticipants []uint64 = common_globals.RemoveDuplicates(participants)
	var newOwner uint64
	for _, participant := range uniqueParticipants {
		if participant != gathering.OwnerPID.Value() {
			newOwner = participant
			break
		}
	}

	// * We couldn't find a new owner, so we unregister the gathering
	if newOwner == 0 {
		nexError = UnregisterGathering(db, gathering.ID.Value)
		if nexError != nil {
			return nexError
		}

		category := notifications.NotificationCategories.GatheringUnregistered
		subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID().Copy().(*types.PID)
		oEvent.Type.Value = notifications.BuildNotificationType(category, subtype)
		oEvent.Param1.Value = gathering.ID.Value

		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, uniqueParticipants)
		return nil
	}

	// * Set the new owner
	gathering.OwnerPID = types.NewPID(newOwner)

	nexError = UpdateSessionHost(db, gathering.ID.Value, gathering.OwnerPID, gathering.HostPID)
	if nexError != nil {
		return nexError
	}

	category := notifications.NotificationCategories.OwnershipChanged
	subtype := notifications.NotificationSubTypes.OwnershipChanged.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(*types.PID)
	oEvent.Type.Value = notifications.BuildNotificationType(category, subtype)
	oEvent.Param1.Value = gathering.ID.Value
	oEvent.Param2.Value = uint32(newOwner) // TODO - This assumes a legacy client. Will not work on the Switch

	// TODO - StrParam doesn't have this value on some servers
	// * https://github.com/kinnay/NintendoClients/issues/101
	// * unixTime := time.Now()
	// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, uniqueParticipants)
	return nil
}
