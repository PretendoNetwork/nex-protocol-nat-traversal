package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetPersistentGatheringParticipationCount gets the number of times a user has participated on a given persistent gathering
func GetPersistentGatheringParticipationCount(manager *common_globals.MatchmakingManager, gatheringID uint32, sourcePID uint64) (uint32, *nex.Error) {
	var persistentGatheringParticipationCount uint32
	err := manager.Database.QueryRow(`SELECT
		cp.participation_count
		FROM matchmaking.community_participations AS cp
		WHERE
		cp.user_pid=$1 AND
		cp.gathering_id=$2`,
		sourcePID,
		gatheringID,
	).Scan(&persistentGatheringParticipationCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return persistentGatheringParticipationCount, nil
}
