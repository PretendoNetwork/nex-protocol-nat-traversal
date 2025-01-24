package database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

// EndGatheringParticipation ends the participation of a connection within a gathering and performs any additional handling required
func EndGatheringParticipation(manager *common_globals.MatchmakingManager, gatheringID uint32, connection *nex.PRUDPConnection, message string) *nex.Error {
	gathering, _, participants, _, nexError := FindGatheringByID(manager, gatheringID)
	if nexError != nil {
		return nexError
	}

	// TODO - Is this the right error?
	if !slices.Contains(participants, uint64(connection.PID())) {
		return nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "change_error")
	}

	newParticipants, nexError := RemoveParticipantFromGathering(manager, gatheringID, uint64(connection.PID()))
	if nexError != nil {
		return nexError
	}

	nexError = tracking.LogLeaveGathering(manager.Database, connection.PID(), gatheringID, newParticipants)
	if nexError != nil {
		return nexError
	}

	// * If the gathering is a persistent gathering and allows zero users, only remove the participant from the gathering
	if uint32(gathering.Flags) & (match_making.GatheringFlags.PersistentGathering | match_making.GatheringFlags.PersistentGatheringAllowZeroUsers) != 0 {
		return nil
	}

	if len(newParticipants) == 0 {
		// * There are no more participants, so we just unregister the gathering
		return UnregisterGathering(manager, connection.PID(), gatheringID)
	}

	var ownerPID uint64 = uint64(gathering.OwnerPID)
	if connection.PID().Equals(gathering.OwnerPID) {
		// * This flag tells the server to change the matchmake session owner if they disconnect
		// * If the flag is not set, delete the session
		// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if uint32(gathering.Flags) & match_making.GatheringFlags.DisconnectChangeOwner == 0 {
			nexError = UnregisterGathering(manager, connection.PID(), gatheringID)
			if nexError != nil {
				return nexError
			}

			category := notifications.NotificationCategories.GatheringUnregistered
			subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = connection.PID().Copy().(types.PID)
			oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
			oEvent.Param1 = types.NewUInt32(gatheringID)

			common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, common_globals.RemoveDuplicates(newParticipants))

			return nil
		}

		ownerPID, nexError = MigrateGatheringOwnership(manager, connection, gathering, newParticipants)
		if nexError != nil {
			return nexError
		}
	}

	// * If the host has disconnected, set the owner as the new host. We can guarantee that the ownerPID is not zero,
	// * since otherwise the gathering would have been unregistered by MigrateGatheringOwnership
	if connection.PID().Equals(gathering.HostPID) && ownerPID != 0 {
		nexError = UpdateSessionHost(manager, gatheringID, types.NewPID(ownerPID), types.NewPID(ownerPID))
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			return nexError
		}

		category := notifications.NotificationCategories.HostChanged
		subtype := notifications.NotificationSubTypes.HostChanged.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID().Copy().(types.PID)
		oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1 = types.NewUInt32(gatheringID)

		// TODO - Should the notification actually be sent to all participants?
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, common_globals.RemoveDuplicates(participants))

		nexError = tracking.LogChangeHost(manager.Database, connection.PID(), gatheringID, gathering.HostPID, types.NewPID(ownerPID))
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			return nexError
		}
	}

	category := notifications.NotificationCategories.Participation
	subtype := notifications.NotificationSubTypes.Participation.Ended

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(types.PID)
	oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = types.NewUInt32(gatheringID)
	oEvent.Param2 = types.NewUInt32(uint32((connection.PID()))) // TODO - This assumes a legacy client. Will not work on the Switch
	oEvent.StrParam = types.NewString(message)

	var participationEndedTargets []uint64

	// * When the VerboseParticipants or VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
	if uint32(gathering.Flags) & (match_making.GatheringFlags.VerboseParticipants | match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
		participationEndedTargets = common_globals.RemoveDuplicates(participants)
	} else {
		participationEndedTargets = []uint64{uint64(gathering.OwnerPID)}
	}

	common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participationEndedTargets)

	return nil
}
