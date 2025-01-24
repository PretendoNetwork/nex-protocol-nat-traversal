package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetSimpleCommunities returns a slice of SimpleCommunity using information from the given gathering IDs
func GetSimpleCommunities(manager *common_globals.MatchmakingManager, gatheringIDList []uint32) ([]match_making_types.SimpleCommunity, *nex.Error) {
	simpleCommunities := make([]match_making_types.SimpleCommunity, 0)

	rows, err := manager.Database.Query(`SELECT
		pg.id
		FROM matchmaking.persistent_gatherings AS pg
		INNER JOIN matchmaking.gatherings AS g ON g.id = pg.id
		WHERE
		g.registered=true AND
		g.type='PersistentGathering' AND
		pg.id=ANY($1)`,
		pqextended.Array(gatheringIDList),
	)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		resultSimpleCommunity := match_making_types.NewSimpleCommunity()
		var resultMatchmakeSessionCount uint32

		err = rows.Scan(
			&resultSimpleCommunity.GatheringID,
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		resultMatchmakeSessionCount, nexError := GetPersistentGatheringSessionCount(manager, uint32(resultSimpleCommunity.GatheringID))
		if nexError != nil {
			common_globals.Logger.Critical(nexError.Error())
			continue
		}
		resultSimpleCommunity.MatchmakeSessionCount = types.NewUInt32(resultMatchmakeSessionCount)

		simpleCommunities = append(simpleCommunities, resultSimpleCommunity)
	}

	rows.Close()

	return simpleCommunities, nil
}
