package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetCreatedPersistentGatherings returns the number of active persistent gatherings that a given PID owns
func GetCreatedPersistentGatherings(manager *common_globals.MatchmakingManager, ownerPID types.PID) (int, *nex.Error) {
	var createdPersistentGatherings int
	err := manager.Database.QueryRow(`SELECT
		COUNT(pg.id)
		FROM matchmaking.persistent_gatherings AS pg
		INNER JOIN matchmaking.gatherings AS g ON g.id = pg.id
		WHERE
		g.registered=true AND
		g.owner_pid=$1`, ownerPID).Scan(&createdPersistentGatherings)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return createdPersistentGatherings, nil
}
