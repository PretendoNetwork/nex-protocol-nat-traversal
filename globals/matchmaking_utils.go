package common_globals

import (
	"sync"
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	"golang.org/x/exp/slices"
)

var sessions map[uint32]*CommonMatchmakeSession
var sessionsMutex = sync.RWMutex{}

var onSessionCreatedHandlers []func(gid uint32)
var onSessionDeletedHandlers []func(gid uint32)
var onPlayerJoinSessionHandlers []func(gid uint32, cid uint32)
var onPlayerLeaveSessionHandlers []func(gid uint32, cid uint32, gracefully bool)

var SessionManagementDebugLog = false

func MakeSessions() {
	sessions = make(map[uint32]*CommonMatchmakeSession)
}

// GetSession returns a session using the gathering ID.
func GetSession(gatheringID uint32) (*CommonMatchmakeSession, bool) {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	ret, ok := sessions[gatheringID]
	return ret, ok
}

// EachSession runs a callback for every session until it returns true
func EachSession(callback func(index uint32, value *CommonMatchmakeSession) bool) bool {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	for i, value := range sessions {
		if callback(i, value) {
			return true
		}
	}

	return false
}

// OnSessionCreated sets a callback that will run just after a session is created
func OnSessionCreated(handler func(gid uint32)) {
	onSessionCreatedHandlers = append(onSessionCreatedHandlers, handler)
}

// OnSessionDeleted sets a callback that will run just before a session is deleted
func OnSessionDeleted(handler func(gid uint32)) {
	onSessionDeletedHandlers = append(onSessionDeletedHandlers, handler)
}

// OnPlayerJoinSession sets a callback that will run just after a connection joins a session
func OnPlayerJoinSession(handler func(gid uint32, cid uint32)) {
	onPlayerJoinSessionHandlers = append(onPlayerJoinSessionHandlers, handler)
}

// OnPlayerLeaveSession sets a callback that will run just before a connection leaves a session
func OnPlayerLeaveSession(handler func(gid uint32, cid uint32, gracefully bool)) {
	onPlayerLeaveSessionHandlers = append(onPlayerLeaveSessionHandlers, handler)
}

// GetAvailableGatheringID returns a gathering ID which doesn't belong to any session
// Returns 0 if no IDs are available (math.MaxUint32 has been reached)
func GetAvailableGatheringID() uint32 {
	return CurrentGatheringID.Next()
}

func findOtherConnectionIDImpl(excludedConnectionID uint32, gatheringID uint32) uint32 {
	var otherConnectionID uint32 = 0
	if session, ok := sessions[gatheringID]; ok {
		session.ConnectionIDs.Each(func(_ int, connectionID uint32) bool {
			if connectionID != excludedConnectionID {
				otherConnectionID = connectionID
				return true
			}

			return false
		})
	}

	return otherConnectionID
}

// FindOtherConnectionID searches a connection ID on the session that isn't the given one
// Returns 0 if no connection ID could be found
func FindOtherConnectionID(excludedConnectionID uint32, gatheringID uint32) uint32 {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	return findOtherConnectionIDImpl(excludedConnectionID, gatheringID)
}

func removeSessionImpl(connection *nex.PRUDPConnection, gathering uint32) {
	session, ok := sessions[gathering]
	if !ok {
		return
	}
	if session.ConnectionIDs.Size() != 0 {
		endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
		server := endpoint.Server

		ownerPID := session.GameMatchmakeSession.Gathering.OwnerPID

		category := notifications.NotificationCategories.GatheringUnregistered
		subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = ownerPID
		oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1 = types.NewPrimitiveU32(gathering)

		stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

		oEvent.WriteTo(stream)

		rmcRequest := nex.NewRMCRequest(endpoint)
		rmcRequest.ProtocolID = notifications.ProtocolID
		rmcRequest.CallID = CurrentMatchmakingCallID.Next()
		rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
		rmcRequest.Parameters = stream.Bytes()

		rmcRequestBytes := rmcRequest.Bytes()

		session.ConnectionIDs.Each(func(_ int, connectionID uint32) bool {
			target := endpoint.FindConnectionByID(connectionID)
			if target == nil {
				Logger.Warning("Connection not found")
				return false
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

			return false
		})
	}
	
	if SessionManagementDebugLog {
		globals.Logger.Infof("GID %d: Deleted", gathering)
	}

	for _, handler := range onSessionDeletedHandlers {
		handler(gathering)
	}

	delete(sessions, gathering)
}

func RemoveSession(connection *nex.PRUDPConnection, gathering uint32) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	removeSessionImpl(connection, gathering)
}

// RemoveConnectionIDFromSession removes a PRUDP connection from the session
func removeConnectionIDFromSessionImpl(connection *nex.PRUDPConnection, gathering uint32, gracefully bool) {
	session, ok := sessions[gathering]
	if !ok {
		return
	}

	for _, handler := range onPlayerLeaveSessionHandlers {
		handler(gathering, connection.ID, gracefully)
	}

	session.ConnectionIDs.DeleteAll(connection.ID)

	ownerPID := session.GameMatchmakeSession.Gathering.OwnerPID
	lenParticipants := session.ConnectionIDs.Size()

	if SessionManagementDebugLog {
		var grace string
		if gracefully {
			grace = "gracefully"
		} else {
			grace = "ungracefully"
		}
		globals.Logger.Infof("GID %d: Removed PID %d %s", gathering, connection.PID().Value(), grace)
	}

	// * If there are no more participants, remove the session
	if lenParticipants == 0 {
		removeSessionImpl(connection, gathering)
		return
	}

	// * If the owner is the one being removed...
	if ownerPID.Equals(connection.PID()) {
		// * This flag tells the server to change the matchmake session owner if they disconnect
		// * If the flag is not set, delete the session
		// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
		if session.GameMatchmakeSession.Gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) == 0 {
			removeSessionImpl(connection, gathering)
			return
		} else {
			changeSessionOwnerImpl(connection, gathering, true)
		}
	}

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	category := notifications.NotificationCategories.Participation

	var subtype uint32
	if gracefully {
		subtype = notifications.NotificationSubTypes.Participation.Ended
	} else {
		subtype = notifications.NotificationSubTypes.Participation.Disconnected
	}

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID()
	oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = types.NewPrimitiveU32(gathering)
	oEvent.Param2 = types.NewPrimitiveU32(connection.PID().LegacyValue()) // TODO - This assumes a legacy client. This won't work on the Switch

	stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(endpoint)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = CurrentMatchmakingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	target := endpoint.FindConnectionByPID(ownerPID.Value())
	if target == nil {
		Logger.Warning("Target connection not found")
		return
	}

	var messagePacket nex.PRUDPPacketInterface

	if connection.DefaultPRUDPVersion == 0 {
		messagePacket, _ = nex.NewPRUDPPacketV0(server, target, nil)
	} else {
		messagePacket, _ = nex.NewPRUDPPacketV1(server, target, nil)
	}

	messagePacket.SetType(constants.DataPacket)
	messagePacket.AddFlag(constants.PacketFlagNeedsAck)
	messagePacket.AddFlag(constants.PacketFlagReliable)
	messagePacket.SetSourceVirtualPortStreamType(connection.StreamType)
	messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
	messagePacket.SetDestinationVirtualPortStreamType(connection.StreamType)
	messagePacket.SetDestinationVirtualPortStreamID(connection.StreamID)
	messagePacket.SetPayload(rmcRequestBytes)

	server.Send(messagePacket)

	// * Update the participation count with the new connection ID count
	session.GameMatchmakeSession.ParticipationCount.Value = uint32(session.ConnectionIDs.Size())
}

func RemoveConnectionIDFromSession(connection *nex.PRUDPConnection, gathering uint32, gracefully bool) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	removeConnectionIDFromSessionImpl(connection, gathering, gracefully)
}

// FindConnectionSession searches for session the given connection ID is connected to
func findConnectionSessionImpl(id uint32) uint32 {
	for gatheringID := range sessions {
		if sessions[gatheringID].ConnectionIDs.Has(id) {
			return gatheringID
		}
	}

	return 0
}

func FindConnectionSession(id uint32) uint32 {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	return findConnectionSessionImpl(id)
}

// RemoveConnectionFromAllsessions removes a connection from every session
func RemoveConnectionFromAllSessions(connection *nex.PRUDPConnection) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	// * Keep checking until no session is found
	for gid := findConnectionSessionImpl(connection.ID); gid != 0; {

		removeConnectionIDFromSessionImpl(connection, gid, false)

		gid = findConnectionSessionImpl(connection.ID)
	}
}

// CreateSessionByMatchmakeSession creates a gathering from a MatchmakeSession
func CreateSessionByMatchmakeSession(matchmakeSession *match_making_types.MatchmakeSession, searchMatchmakeSession *match_making_types.MatchmakeSession, hostPID *types.PID) (*CommonMatchmakeSession, *nex.Error) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	
	sessionIndex := GetAvailableGatheringID()
	if sessionIndex == 0 {
		sessionIndex = GetAvailableGatheringID() // * Skip to index 1
	}

	session := CommonMatchmakeSession{
		SearchMatchmakeSession: searchMatchmakeSession,
		GameMatchmakeSession:   matchmakeSession,
		ConnectionIDs:          nex.NewMutexSlice[uint32](),
	}

	session.GameMatchmakeSession.Gathering.ID = types.NewPrimitiveU32(sessionIndex)
	session.GameMatchmakeSession.Gathering.OwnerPID = hostPID
	session.GameMatchmakeSession.Gathering.HostPID = hostPID

	session.GameMatchmakeSession.StartedTime = types.NewDateTime(0).Now()
	session.GameMatchmakeSession.SessionKey = types.NewBuffer(make([]byte, 32))

	rand.Read(session.GameMatchmakeSession.SessionKey.Value)

	SR := types.NewVariant()
	SR.TypeID = types.NewPrimitiveU8(3)
	SR.Type = types.NewPrimitiveBool(true)

	GIR := types.NewVariant()
	GIR.TypeID = types.NewPrimitiveU8(1)
	GIR.Type = types.NewPrimitiveS64(3)

	session.GameMatchmakeSession.MatchmakeParam.Params.Set(types.NewString("@SR"), SR)
	session.GameMatchmakeSession.MatchmakeParam.Params.Set(types.NewString("@GIR"), GIR)

	sessions[sessionIndex] = &session

	if SessionManagementDebugLog {
		globals.Logger.Infof("GID %d: Created", sessionIndex)
	}

	for _, handler := range onSessionCreatedHandlers {
		handler(session.GameMatchmakeSession.ID.Value)
	}

	return sessions[sessionIndex], nil
}

// isSessionHostConnected checks if the current session host is connected
func isSessionHostConnected(session *CommonMatchmakeSession, endpoint *nex.PRUDPEndPoint) bool {
	return session.ConnectionIDs.Each(func(_ int, connectionID uint32) bool {
		target := endpoint.FindConnectionByID(connectionID)
		if target == nil {
			return false
		}
		return session.GameMatchmakeSession.HostPID.Equals(target.PID())
	})
}

// FindSessionByMatchmakeSession finds a gathering that matches with a MatchmakeSession
func FindSessionByMatchmakeSession(connection *nex.PRUDPConnection, searchMatchmakeSession *match_making_types.MatchmakeSession) uint32 {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	// * This portion finds any sessions that match the search session
	// * It does not care about anything beyond that, such as if the match is already full
	// * This is handled below
	candidateSessionIndexes := make([]uint32, 0, len(sessions))
	for index, session := range sessions {
		if !session.SearchMatchmakeSession.Equals(searchMatchmakeSession) {
			continue
		}
		// * Do not find the session if the host is not currently connected
		if !isSessionHostConnected(session, connection.Endpoint().(*nex.PRUDPEndPoint)) {
			continue
		}

		// * Do not find the room if the requesting connection is the host. This means
		// * the host was disconnected but the room host PID wasn't updated yet by the rest of
		// * the clients. The host suddenly being available again causes issues.
		if (session.GameMatchmakeSession.HostPID.Equals(connection.PID())) {
			continue
		}

		candidateSessionIndexes = append(candidateSessionIndexes, index)
	}

	// TODO - This whole section assumes legacy clients. None of it will work on the Switch
	var friendList []uint32
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck, ok := sessions[sessionIndex]
		if !ok {
			continue
		}

		if sessionToCheck.ConnectionIDs.Size() >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants.Value) {
			continue
		}

		if !sessionToCheck.GameMatchmakeSession.OpenParticipation.Value {
			continue
		}

		// * If the session only allows friends, check if the owner is in the friend list of the PID
		// TODO - Is this a flag or a constant?
		if sessionToCheck.GameMatchmakeSession.ParticipationPolicy.Value == 98 {
			if GetUserFriendPIDsHandler == nil {
				Logger.Warning("Missing GetUserFriendPIDsHandler!")
				continue
			}

			if len(friendList) == 0 {
				friendList = GetUserFriendPIDsHandler(connection.PID().LegacyValue()) // TODO - This grpc method needs to support the Switch
			}

			if !slices.Contains(friendList, sessionToCheck.GameMatchmakeSession.OwnerPID.LegacyValue()) {
				continue
			}
		}

		return sessionIndex // * Found a match
	}

	return 0
}

// FindSessionsByMatchmakeSessionSearchCriterias finds a gathering that matches with the given search criteria
func FindSessionsByMatchmakeSessionSearchCriterias(connection *nex.PRUDPConnection, searchCriterias []*match_making_types.MatchmakeSessionSearchCriteria, gameSpecificChecks func(searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, matchmakeSession *match_making_types.MatchmakeSession) bool) []*CommonMatchmakeSession {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()
	
	candidateSessions := make([]*CommonMatchmakeSession, 0, len(sessions))

	// TODO - This whole section assumes legacy clients. None of it will work on the Switch
	var friendList []uint32
	for _, session := range sessions {
		// * Do not find the session if the host is not currently connected
		if !isSessionHostConnected(session, connection.Endpoint().(*nex.PRUDPEndPoint)) {
			continue
		}

		// * Do not find the room if the requesting connection is the host. This means
		// * the host was disconnected but the room host PID wasn't updated yet by the rest of
		// * the clients. The host suddenly being available again causes issues.
		if (session.GameMatchmakeSession.HostPID.Equals(connection.PID())) {
			continue
		}

		for _, criteria := range searchCriterias {
			// * Check things like game specific attributes
			if gameSpecificChecks != nil {
				if !gameSpecificChecks(criteria, session.GameMatchmakeSession) {
					continue
				}
			} else {
				if !compareAttributesSearchCriteria(session.GameMatchmakeSession.Attributes.Slice(), criteria.Attribs.Slice()) {
					continue
				}
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.MaximumParticipants.Value, criteria.MaxParticipants.Value) {
				continue
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.MinimumParticipants.Value, criteria.MinParticipants.Value) {
				continue
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.MatchmakeSystemType.Value, criteria.MatchmakeSystemType.Value) {
				continue
			}

			if !compareSearchCriteria(session.GameMatchmakeSession.GameMode.Value, criteria.GameMode.Value) {
				continue
			}

			if session.ConnectionIDs.Size() >= int(session.GameMatchmakeSession.MaximumParticipants.Value) {
				continue
			}

			if !session.GameMatchmakeSession.OpenParticipation.Value {
				continue
			}

			// * If the session only allows friends, check if the owner is in the friend list of the PID
			// TODO - Is this a flag or a constant?
			if session.GameMatchmakeSession.ParticipationPolicy.Value == 98 {
				if GetUserFriendPIDsHandler == nil {
					Logger.Warning("Missing GetUserFriendPIDsHandler!")
					continue
				}

				if len(friendList) == 0 {
					friendList = GetUserFriendPIDsHandler(connection.PID().LegacyValue()) // TODO - Support the Switch
				}

				if !slices.Contains(friendList, session.GameMatchmakeSession.OwnerPID.LegacyValue()) {
					continue
				}
			}

			candidateSessions = append(candidateSessions, session)

			// * We don't have to compare with other search criterias
			break
		}
	}

	return candidateSessions
}

func compareAttributesSearchCriteria(original []*types.PrimitiveU32, search []*types.String) bool {
	if len(original) != len(search) {
		return false
	}

	for index, originalAttribute := range original {
		searchAttribute := search[index]

		if !compareSearchCriteria(originalAttribute.Value, searchAttribute.Value) {
			return false
		}
	}

	return true
}

func compareSearchCriteria[T ~uint16 | ~uint32](original T, search string) bool {
	if search == "" { // * Accept any value
		return true
	}

	before, after, found := strings.Cut(search, ",")
	if found {
		min, err := strconv.ParseUint(before, 10, 64)
		if err != nil {
			return false
		}

		max, err := strconv.ParseUint(after, 10, 64)
		if err != nil {
			return false
		}

		return min <= uint64(original) && max >= uint64(original)
	} else {
		searchNum, err := strconv.ParseUint(before, 10, 64)
		if err != nil {
			return false
		}

		return searchNum == uint64(original)
	}
}

// AddPlayersToSession updates the given sessions state to include the provided connection IDs
// Returns a NEX error code if failed
func AddPlayersToSession(session *CommonMatchmakeSession, connectionIDs []uint32, initiatingConnection *nex.PRUDPConnection, joinMessage string) *nex.Error {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	if (session.ConnectionIDs.Size() + len(connectionIDs)) > int(session.GameMatchmakeSession.Gathering.MaximumParticipants.Value) {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionFull, fmt.Sprintf("Gathering %d is full", session.GameMatchmakeSession.Gathering.ID))
	}

	// * TOCTOU, just in case
	_, ok := sessions[session.GameMatchmakeSession.Gathering.ID.Value]
	if !ok {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	endpoint := initiatingConnection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	for _, connectedID := range connectionIDs {
		if session.ConnectionIDs.Has(connectedID) {
			return nex.NewError(nex.ResultCodes.RendezVous.AlreadyParticipatedGathering, fmt.Sprintf("Connection ID %d is already in gathering %d", connectedID, session.GameMatchmakeSession.Gathering.ID))
		}

		session.ConnectionIDs.Add(connectedID)

		if SessionManagementDebugLog {
			conn := endpoint.FindConnectionByID(connectedID)
			globals.Logger.Infof("GID %d: Added PID %d", session.GameMatchmakeSession.Gathering.ID.Value, conn.PID().Value())
		}

		// * Update the participation count with the new connection ID count
		session.GameMatchmakeSession.ParticipationCount.Value = uint32(session.ConnectionIDs.Size())

		for _, handler := range onPlayerJoinSessionHandlers {
			handler(session.GameMatchmakeSession.ID.Value, connectedID)
		}
	}

	target := endpoint.FindConnectionByPID(session.GameMatchmakeSession.OwnerPID.Value())
	if target != nil {
		notificationCategory := notifications.NotificationCategories.Participation
		notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = initiatingConnection.PID()
		oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
		oEvent.Param1 = session.GameMatchmakeSession.ID.Copy().(*types.PrimitiveU32)
		oEvent.Param2 = types.NewPrimitiveU32(target.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch
		oEvent.StrParam = types.NewString(joinMessage)
		oEvent.Param3 = types.NewPrimitiveU32(uint32(len(connectionIDs)))

		notificationStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

		oEvent.WriteTo(notificationStream)

		notificationRequest := nex.NewRMCRequest(endpoint)
		notificationRequest.ProtocolID = notifications.ProtocolID
		notificationRequest.CallID = CurrentMatchmakingCallID.Next()
		notificationRequest.MethodID = notifications.MethodProcessNotificationEvent
		notificationRequest.Parameters = notificationStream.Bytes()

		notificationRequestBytes := notificationRequest.Bytes()

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
		messagePacket.SetPayload(notificationRequestBytes)

		server.Send(messagePacket)
	}	

	// * This appears to be correct. Tri-Force Heroes uses 3.9.0,
	// * and has issues if these notifications are sent.
	// * Minecraft, however, requires these to be sent
	// TODO - Check other games both pre and post 3.10.0 and validate
	if server.LibraryVersions.MatchMaking.GreaterOrEqual("3.10.0") {
		session.ConnectionIDs.Each(func(_ int, connectionID uint32) bool {
			target := endpoint.FindConnectionByID(connectionID)
			if target == nil {
				// TODO - Error here?
				Logger.Warning("Player not found")
				return false
			}

			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = initiatingConnection.PID()
			oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
			oEvent.Param1 = session.GameMatchmakeSession.ID.Copy().(*types.PrimitiveU32)
			oEvent.Param2 = types.NewPrimitiveU32(target.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch
			oEvent.StrParam = types.NewString(joinMessage)
			oEvent.Param3 = types.NewPrimitiveU32(uint32(len(connectionIDs)))

			notificationStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

			oEvent.WriteTo(notificationStream)

			notificationRequest := nex.NewRMCRequest(endpoint)
			notificationRequest.ProtocolID = notifications.ProtocolID
			notificationRequest.CallID = CurrentMatchmakingCallID.Next()
			notificationRequest.MethodID = notifications.MethodProcessNotificationEvent
			notificationRequest.Parameters = notificationStream.Bytes()

			notificationRequestBytes := notificationRequest.Bytes()

			var messagePacket nex.PRUDPPacketInterface

			if target.DefaultPRUDPVersion == 0 {
				messagePacket, _ = nex.NewPRUDPPacketV0(server, initiatingConnection, nil)
			} else {
				messagePacket, _ = nex.NewPRUDPPacketV1(server, initiatingConnection, nil)
			}

			messagePacket.SetType(constants.DataPacket)
			messagePacket.AddFlag(constants.PacketFlagNeedsAck)
			messagePacket.AddFlag(constants.PacketFlagReliable)
			messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
			messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
			messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
			messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
			messagePacket.SetPayload(notificationRequestBytes)

			server.Send(messagePacket)

			return false
		})

		notificationCategory := notifications.NotificationCategories.Participation
		notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = initiatingConnection.PID()
		oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
		oEvent.Param1 = session.GameMatchmakeSession.ID
		oEvent.Param2 = types.NewPrimitiveU32(initiatingConnection.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch
		oEvent.StrParam = types.NewString(joinMessage)
		oEvent.Param3 = types.NewPrimitiveU32(uint32(len(connectionIDs)))

		notificationStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

		oEvent.WriteTo(notificationStream)

		notificationRequest := nex.NewRMCRequest(endpoint)
		notificationRequest.ProtocolID = notifications.ProtocolID
		notificationRequest.CallID = CurrentMatchmakingCallID.Next()
		notificationRequest.MethodID = notifications.MethodProcessNotificationEvent
		notificationRequest.Parameters = notificationStream.Bytes()

		notificationRequestBytes := notificationRequest.Bytes()

		var messagePacket nex.PRUDPPacketInterface

		if initiatingConnection.DefaultPRUDPVersion == 0 {
			messagePacket, _ = nex.NewPRUDPPacketV0(server, initiatingConnection, nil)
		} else {
			messagePacket, _ = nex.NewPRUDPPacketV1(server, initiatingConnection, nil)
		}

		messagePacket.SetType(constants.DataPacket)
		messagePacket.AddFlag(constants.PacketFlagNeedsAck)
		messagePacket.AddFlag(constants.PacketFlagReliable)
		messagePacket.SetSourceVirtualPortStreamType(initiatingConnection.StreamType)
		messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
		messagePacket.SetDestinationVirtualPortStreamType(initiatingConnection.StreamType)
		messagePacket.SetDestinationVirtualPortStreamID(initiatingConnection.StreamID)
		messagePacket.SetPayload(notificationRequestBytes)

		server.Send(messagePacket)

		target := endpoint.FindConnectionByPID(session.GameMatchmakeSession.Gathering.OwnerPID.Value())
		if target == nil {
			// TODO - Error here?
			Logger.Warning("Player not found")
			return nil
		}

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
		messagePacket.SetPayload(notificationRequestBytes)

		server.Send(messagePacket)
	}

	return nil
}

// ChangeSessionOwner changes the session owner to a different connection
func changeSessionOwnerImpl(currentOwner *nex.PRUDPConnection, gathering uint32, isLeaving bool) {
	endpoint := currentOwner.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server
	session, ok := sessions[gathering]
	if !ok {
		return
	}

	var newOwner *nex.PRUDPConnection

	newOwnerConnectionID := findOtherConnectionIDImpl(currentOwner.ID, gathering)
	if newOwnerConnectionID != 0 {
		newOwner = endpoint.FindConnectionByID(newOwnerConnectionID)
		if newOwner == nil {
			Logger.Warning("Other connection not found")
			return
		}

		if SessionManagementDebugLog {
			globals.Logger.Infof("GID %d: ChangeSessionOwner OWNER from PID %d to PID %d", gathering, currentOwner.PID().Value(), newOwner.PID().Value())
		}
	
		session.GameMatchmakeSession.Gathering.OwnerPID = newOwner.PID()
	} else {
		return
	}

	category := notifications.NotificationCategories.OwnershipChanged
	subtype := notifications.NotificationSubTypes.OwnershipChanged.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = currentOwner.PID()
	oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = types.NewPrimitiveU32(gathering)
	oEvent.Param2 = types.NewPrimitiveU32(newOwner.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch

	// TODO - StrParam doesn't have this value on some servers
	// * https://github.com/kinnay/NintendoClients/issues/101
	// * unixTime := time.Now()
	// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	oEvent.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(endpoint)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = CurrentMatchmakingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	session.ConnectionIDs.Each(func(_ int, connectionID uint32) bool {
		target := endpoint.FindConnectionByID(connectionID)
		if target == nil {
			Logger.Warning("Connection not found")
			return false
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

		return false
	})
}

func ChangeSessionOwner(currentOwner *nex.PRUDPConnection, gathering uint32, isLeaving bool) {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	changeSessionOwnerImpl(currentOwner, gathering, isLeaving)
}
