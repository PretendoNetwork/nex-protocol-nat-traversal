package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// RegisterGathering registers a new gathering on the database. No participants are added
func RegisterGathering(manager *common_globals.MatchmakingManager, ownerPID types.PID, hostPID types.PID, gathering *match_making_types.Gathering, gatheringType string) (types.DateTime, *nex.Error) {
	startedTime := types.NewDateTime(0).Now()
	var gatheringID uint32

	err := manager.Database.QueryRow(`INSERT INTO matchmaking.gatherings (
		owner_pid,
		host_pid,
		min_participants,
		max_participants,
		participation_policy,
		policy_argument,
		flags,
		state,
		description,
		type,
		started_time
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		$10,
		$11
	) RETURNING id`,
		ownerPID,
		hostPID,
		gathering.MinimumParticipants,
		gathering.MaximumParticipants,
		gathering.ParticipationPolicy,
		gathering.PolicyArgument,
		gathering.Flags,
		gathering.State,
		gathering.Description,
		gatheringType,
		startedTime.Standard(),
	).Scan(&gatheringID)
	if err != nil {
		return types.NewDateTime(0), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	gathering.ID = types.NewUInt32(gatheringID)

	nexError := tracking.LogRegisterGathering(manager.Database, ownerPID, uint32(gathering.ID))
	if nexError != nil {
		return types.NewDateTime(0), nexError
	}

	gathering.OwnerPID = ownerPID.Copy().(types.PID)
	gathering.HostPID = hostPID.Copy().(types.PID)

	return startedTime, nil
}
