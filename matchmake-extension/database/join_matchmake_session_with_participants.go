package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/tracking"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// JoinMatchmakeSessionWithParticipants joins participants into a gathering. Returns the new number of participants
func JoinMatchmakeSessionWithParticipants(manager *common_globals.MatchmakingManager, matchmakeSession match_making_types.MatchmakeSession, connection *nex.PRUDPConnection, additionalParticipants []types.PID, joinMessage string, joinMatchmakeSessionBehavior constants.JoinMatchmakeSessionBehavior) (uint32, *nex.Error) {
	newParticipants, nexError := match_making_database.JoinGatheringWithParticipants(manager, uint32(matchmakeSession.ID), connection, additionalParticipants, joinMessage, joinMatchmakeSessionBehavior)
	if nexError != nil {
		return 0, nexError
	}

	// TODO - Should we return the error in these cases?
	if uint32(matchmakeSession.MatchmakeSystemType) == uint32(constants.MatchmakeSystemTypePersistentGathering) { // * Attached to a persistent gathering
		persistentGatheringID := uint32(matchmakeSession.Attributes[0])
		participantList := append(additionalParticipants, connection.PID())
		for _, participant := range participantList {
			_, nexError = GetPersistentGatheringByID(manager, participant, persistentGatheringID)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
				continue
			}

			participationCount, nexError := UpdatePersistentGatheringParticipationCount(manager, participant, persistentGatheringID)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
				continue
			}

			nexError = tracking.LogParticipateCommunity(manager.Database, participant, persistentGatheringID, uint32(matchmakeSession.ID), participationCount)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
				continue
			}
		}

	}

	return newParticipants, nil
}
