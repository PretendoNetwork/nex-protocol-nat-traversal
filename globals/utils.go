package common_globals

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_constants "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

var OutgoingCallID *nex.Counter[uint32] = nex.NewCounter[uint32](0)

// DeleteIndex removes a value from a slice with the given index
func DeleteIndex(s []uint32, index int) []uint32 {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

// RemoveDuplicates removes duplicate entries on a slice
func RemoveDuplicates[T comparable](sliceList []T) []T {
	// * Extracted from https://stackoverflow.com/a/66751055
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// CheckValidGathering checks if a Gathering is valid
func CheckValidGathering(gathering match_making_types.Gathering) bool {
	if len(gathering.Description) > 256 {
		return false
	}

	return true
}

// CheckValidMatchmakeSession checks if a MatchmakeSession is valid
func CheckValidMatchmakeSession(matchmakeSession match_making_types.MatchmakeSession) bool {
	if !CheckValidGathering(matchmakeSession.Gathering) {
		return false
	}

	if len(matchmakeSession.Attributes) != 6 {
		return false
	}

	if matchmakeSession.ProgressScore > 100 {
		return false
	}

	if len(matchmakeSession.UserPassword) > 32 {
		return false
	}

	// * Except for UserPassword, all strings must have a length lower than 256
	if len(matchmakeSession.CodeWord) > 256 {
		return false
	}

	// * All buffers must have a length lower than 512
	if len(matchmakeSession.ApplicationBuffer) > 512 {
		return false
	}

	if len(matchmakeSession.SessionKey) > 512 {
		return false
	}

	return true
}

// CheckValidPersistentGathering checks if a PersistentGathering is valid
func CheckValidPersistentGathering(persistentGathering match_making_types.PersistentGathering) bool {
	if !CheckValidGathering(persistentGathering.Gathering) {
		return false
	}

	// * Only allow normal and password-protected community types
	if uint32(persistentGathering.CommunityType) != uint32(match_making_constants.PersistentGatheringTypeOpen) && uint32(persistentGathering.CommunityType) != uint32(match_making_constants.PersistentGatheringTypePasswordLocked) {
		return false
	}

	// * The UserPassword from a MatchmakeSession can be up to 32 characters, assuming the same here
	//
	// TODO - IS this actually the case?
	if len(persistentGathering.Password) > 32 {
		return false
	}

	if len(persistentGathering.Attribs) != 6 {
		return false
	}

	// * All buffers must have a length lower than 512
	if len(persistentGathering.ApplicationBuffer) > 512 {
		return false
	}

	return true
}

// CanJoinMatchmakeSession checks if a PID is allowed to join a matchmake session
func CanJoinMatchmakeSession(manager *MatchmakingManager, pid types.PID, matchmakeSession match_making_types.MatchmakeSession) *nex.Error {
	// TODO - Is this the right error?
	if !matchmakeSession.OpenParticipation {
		return nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	// * Only allow friends
	// TODO - This won't work on Switch!
	if matchmakeSession.ParticipationPolicy == 98 {
		if manager.GetUserFriendPIDs == nil {
			Logger.Warning("Missing GetUserFriendPIDs!")
			return nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		friendList := manager.GetUserFriendPIDs(uint32(pid))
		if !slices.Contains(friendList, uint32(matchmakeSession.OwnerPID)) {
			return nex.NewError(nex.ResultCodes.RendezVous.NotFriend, "change_error")
		}
	}

	return nil
}

// SendNotificationEvent sends a notification event to the specified targets
func SendNotificationEvent(endpoint *nex.PRUDPEndPoint, event notifications_types.NotificationEvent, targets []uint64) {
	server := endpoint.Server
	stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	event.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(endpoint)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = OutgoingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	for _, pid := range targets {
		target := endpoint.FindConnectionByPID(pid)
		if target == nil {
			Logger.Warning("Client not found")
			continue
		}

		var messagePacket nex.PRUDPPacketInterface

		if target.DefaultPRUDPVersion == 0 {
			messagePacket, _ = nex.NewPRUDPPacketV0(server, target, nil)
		} else {
			messagePacket, _ = nex.NewPRUDPPacketV1(server, target, nil)
		}

		messagePacket.SetType(constants.DataPacket)
		messagePacket.AddFlag(constants.PacketFlagNeedsAck)
		messagePacket.AddFlag(constants.PacketFlagReliable)
		messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
		messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
		messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
		messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
		messagePacket.SetPayload(rmcRequestBytes)

		server.Send(messagePacket)
	}
}
