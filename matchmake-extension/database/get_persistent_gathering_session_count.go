package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetPersistentGatheringSessionCount gets the number of active matchmake sessions attached to a given persistent gathering
func GetPersistentGatheringSessionCount(manager *common_globals.MatchmakingManager, gatheringID uint32) (uint32, *nex.Error) {
	var persistentGatheringSessionCount uint32
	err := manager.Database.QueryRow(`SELECT
		COUNT(ms.id)
		FROM matchmaking.matchmake_sessions AS ms
		INNER JOIN matchmaking.gatherings AS g ON ms.id = g.id
		WHERE
		g.registered=true AND
		ms.matchmake_system_type=5 AND
		ms.attribs[1]=$1`, // * matchmake_system_type=5 is only used in matchmake sessions attached to a persistent gathering
		gatheringID,
	).Scan(&persistentGatheringSessionCount)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return persistentGatheringSessionCount, nil
}
