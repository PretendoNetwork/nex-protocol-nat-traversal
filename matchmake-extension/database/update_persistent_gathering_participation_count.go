package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdatePersistentGatheringParticipationCount updates the participation count of a user in a persistent gathering. Returns the participation count
func UpdatePersistentGatheringParticipationCount(manager *common_globals.MatchmakingManager, userPID types.PID, gatheringID uint32) (uint32, *nex.Error) {
	var participationCount uint32
	err := manager.Database.QueryRow(`INSERT INTO matchmaking.community_participations AS cp (
		user_pid,
		gathering_id,
		participation_count
	) VALUES (
		$1,
		$2,
		1
	) ON CONFLICT (user_pid, gathering_id) DO UPDATE SET
	participation_count=cp.participation_count+1 RETURNING participation_count`,
		uint64(userPID),
		gatheringID,
	).Scan(&participationCount)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return participationCount, nil
}
