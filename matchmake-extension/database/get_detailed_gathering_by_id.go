package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
)

// GetDetailedGatheringByID returns a Gathering as an RVType by its gathering ID
func GetDetailedGatheringByID(manager *common_globals.MatchmakingManager, sourcePID uint64, gatheringID uint32) (types.RVType, string, *nex.Error) {
	gathering, gatheringType, participants, startedTime, nexError := match_making_database.FindGatheringByID(manager, gatheringID)
	if nexError != nil {
		return nil, "", nexError
	}

	if gatheringType == "Gathering" {
		return gathering, gatheringType, nil
	}

	if gatheringType == "MatchmakeSession" {
		matchmakeSession, nexError := GetMatchmakeSessionByGathering(manager, manager.Endpoint, gathering, uint32(len(participants)), startedTime)
		if nexError != nil {
			return nil, "", nexError
		}

		// * Scrap session key and user password
		matchmakeSession.SessionKey = make([]byte, 0)
		matchmakeSession.UserPassword = ""

		return matchmakeSession, gatheringType, nil
	}

	if gatheringType == "PersistentGathering" {
		persistentGathering, nexError := GetPersistentGatheringByGathering(manager, gathering, sourcePID)
		if nexError != nil {
			return nil, "", nexError
		}

		matchmakeSessionCount, nexError := GetPersistentGatheringSessionCount(manager, gatheringID)
		if nexError != nil {
			return nil, "", nexError
		}

		participationCount, nexError := GetPersistentGatheringParticipationCount(manager, gatheringID, sourcePID)
		if nexError != nil {
			return nil, "", nexError
		}

		persistentGathering.MatchmakeSessionCount = types.NewUInt32(matchmakeSessionCount)
		persistentGathering.ParticipationCount = types.NewUInt32(participationCount)

		// * Scrap persistent gathering password
		persistentGathering.Password = ""

		return persistentGathering, gatheringType, nil
	}

	return nil, "", nex.NewError(nex.ResultCodes.Core.Exception, "change_error")
}
