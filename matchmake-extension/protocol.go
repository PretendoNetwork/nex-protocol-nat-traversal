package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
)

type CommonProtocol struct {
	endpoint                                         nex.EndpointInterface
	protocol                                         matchmake_extension.Interface
	manager                                          *common_globals.MatchmakingManager
	PersistentGatheringCreationMax                   int
	CanJoinMatchmakeSession                          func(manager *common_globals.MatchmakingManager, pid types.PID, matchmakeSession match_making_types.MatchmakeSession) *nex.Error
	CleanupSearchMatchmakeSession                    func(matchmakeSession *match_making_types.MatchmakeSession)
	CleanupMatchmakeSessionSearchCriterias           func(searchCriterias types.List[match_making_types.MatchmakeSessionSearchCriteria])
	OnAfterOpenParticipation                         func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterCloseParticipation                        func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterCreateMatchmakeSession                    func(packet nex.PacketInterface, anyGathering match_making_types.GatheringHolder, message types.String, participationCount types.UInt16)
	OnAfterGetSimplePlayingSession                   func(packet nex.PacketInterface, listPID types.List[types.PID], includeLoginUser types.Bool)
	OnAfterAutoMatchmakePostpone                     func(packet nex.PacketInterface, anyGathering match_making_types.GatheringHolder, message types.String)
	OnAfterAutoMatchmakeWithParamPostpone            func(packet nex.PacketInterface, autoMatchmakeParam match_making_types.AutoMatchmakeParam)
	OnAfterAutoMatchmakeWithSearchCriteriaPostpone   func(packet nex.PacketInterface, lstSearchCriteria types.List[match_making_types.MatchmakeSessionSearchCriteria], anyGathering match_making_types.GatheringHolder, strMessage types.String)
	OnAfterCreateCommunity                           func(packet nex.PacketInterface, community match_making_types.PersistentGathering, strMessage types.String)
	OnAfterFindCommunityByGatheringID                func(packet nex.PacketInterface, lstGID types.List[types.UInt32])
	OnAfterFindOfficialCommunity                     func(packet nex.PacketInterface, isAvailableOnly types.Bool, resultRange types.ResultRange)
	OnAfterFindCommunityByParticipant                func(packet nex.PacketInterface, pid types.PID, resultRange types.ResultRange)
	OnAfterUpdateProgressScore                       func(packet nex.PacketInterface, gid types.UInt32, progressScore types.UInt8)
	OnAfterCreateMatchmakeSessionWithParam           func(packet nex.PacketInterface, createMatchmakeSessionParam match_making_types.CreateMatchmakeSessionParam)
	OnAfterUpdateApplicationBuffer                   func(packet nex.PacketInterface, gid types.UInt32, applicationBuffer types.Buffer)
	OnAfterJoinMatchmakeSession                      func(packet nex.PacketInterface, gid types.UInt32, strMessage types.String)
	OnAfterJoinMatchmakeSessionWithParam             func(packet nex.PacketInterface, joinMatchmakeSessionParam match_making_types.JoinMatchmakeSessionParam)
	OnAfterModifyCurrentGameAttribute                func(packet nex.PacketInterface, gid types.UInt32, attribIndex types.UInt32, newValue types.UInt32)
	OnAfterBrowseMatchmakeSession                    func(packet nex.PacketInterface, searchCriteria match_making_types.MatchmakeSessionSearchCriteria, resultRange types.ResultRange)
	OnAfterJoinMatchmakeSessionEx                    func(packet nex.PacketInterface, gid types.UInt32, strMessage types.String, dontCareMyBlockList types.Bool, participationCount types.UInt16)
	OnAfterGetSimpleCommunity                        func(packet nex.PacketInterface, gatheringIDList types.List[types.UInt32])
}

// SetDatabase defines the matchmaking manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.MatchmakingManager) {
	var err error

	commonProtocol.manager = manager

	manager.GetDetailedGatheringByID = database.GetDetailedGatheringByID

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.matchmake_sessions (
		id bigserial PRIMARY KEY,
		game_mode bigint,
		attribs bigint[],
		open_participation boolean,
		matchmake_system_type bigint,
		application_buffer bytea,
		flags bigint,
		state bigint,
		progress_score smallint,
		session_key bytea,
		option_zero bigint,
		matchmake_param bytea,
		user_password text,
		refer_gid bigint,
		user_password_enabled boolean,
		system_password_enabled boolean,
		codeword text,
		system_password text NOT NULL DEFAULT ''
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.persistent_gatherings (
		id bigserial PRIMARY KEY,
		community_type bigint,
		password text,
		attribs bigint[],
		application_buffer bytea,
		participation_start_date timestamp,
		participation_end_date timestamp
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.community_participations (
		id bigserial PRIMARY KEY,
		user_pid numeric(10),
		gathering_id bigint,
		participation_count bigint,
		UNIQUE (user_pid, gathering_id)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.participate_community (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(10),
		community_gid bigint,
		gathering_id bigint,
		participation_count bigint
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * In case the server is restarted, unregister any previous matchmake sessions
	_, err = manager.Database.Exec(`UPDATE matchmaking.gatherings SET registered=false WHERE type='MatchmakeSession'`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol matchmake_extension.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
		PersistentGatheringCreationMax: 4, // * Default of 4 active persistent gatherings per user
	}

	protocol.SetHandlerOpenParticipation(commonProtocol.openParticipation)
	protocol.SetHandlerCloseParticipation(commonProtocol.closeParticipation)
	protocol.SetHandlerCreateMatchmakeSession(commonProtocol.createMatchmakeSession)
	protocol.SetHandlerGetSimplePlayingSession(commonProtocol.getSimplePlayingSession)
	protocol.SetHandlerAutoMatchmakePostpone(commonProtocol.autoMatchmakePostpone)
	protocol.SetHandlerAutoMatchmakeWithParamPostpone(commonProtocol.autoMatchmakeWithParamPostpone)
	protocol.SetHandlerAutoMatchmakeWithSearchCriteriaPostpone(commonProtocol.autoMatchmakeWithSearchCriteriaPostpone)
	protocol.SetHandlerCreateCommunity(commonProtocol.createCommunity)
	protocol.SetHandlerFindCommunityByGatheringID(commonProtocol.findCommunityByGatheringID)
	protocol.SetHandlerFindOfficialCommunity(commonProtocol.findOfficialCommunity)
	protocol.SetHandlerFindCommunityByParticipant(commonProtocol.findCommunityByParticipant)
	protocol.SetHandlerUpdateProgressScore(commonProtocol.updateProgressScore)
	protocol.SetHandlerCreateMatchmakeSessionWithParam(commonProtocol.createMatchmakeSessionWithParam)
	protocol.SetHandlerUpdateApplicationBuffer(commonProtocol.updateApplicationBuffer)
	protocol.SetHandlerJoinMatchmakeSession(commonProtocol.joinMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionWithParam(commonProtocol.joinMatchmakeSessionWithParam)
	protocol.SetHandlerModifyCurrentGameAttribute(commonProtocol.modifyCurrentGameAttribute)
	protocol.SetHandlerBrowseMatchmakeSession(commonProtocol.browseMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionEx(commonProtocol.joinMatchmakeSessionEx)
	protocol.SetHandlerGetSimpleCommunity(commonProtocol.getSimpleCommunity)

	return commonProtocol
}
