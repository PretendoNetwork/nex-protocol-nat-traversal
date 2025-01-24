package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetPersistentGatheringsByID gets the persistent gatherings from the given gathering IDs
func GetPersistentGatheringsByID(manager *common_globals.MatchmakingManager, sourcePID types.PID, gatheringIDs []uint32) ([]match_making_types.PersistentGathering, *nex.Error) {
	persistentGatherings := make([]match_making_types.PersistentGathering, 0)
	rows, err := manager.Database.Query(`SELECT
		g.id,
		g.owner_pid,
		g.host_pid,
		g.min_participants,
		g.max_participants,
		g.participation_policy,
		g.policy_argument,
		g.flags,
		g.state,
		g.description,
		pg.community_type,
		pg.password,
		pg.attribs,
		pg.application_buffer,
		pg.participation_start_date,
		pg.participation_end_date
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.persistent_gatherings AS pg ON g.id = pg.id
		WHERE
		g.registered=true AND
		g.type='PersistentGathering' AND
		g.id=ANY($1)`,
		pqextended.Array(gatheringIDs),
	)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for rows.Next() {
		resultPersistentGathering := match_making_types.NewPersistentGathering()
		var resultAttribs []uint32
		var resultParticipationStartDate time.Time
		var resultParticipationEndDate time.Time

		err = rows.Scan(
			&resultPersistentGathering.Gathering.ID,
			&resultPersistentGathering.Gathering.OwnerPID,
			&resultPersistentGathering.Gathering.HostPID,
			&resultPersistentGathering.Gathering.MinimumParticipants,
			&resultPersistentGathering.Gathering.MaximumParticipants,
			&resultPersistentGathering.Gathering.ParticipationPolicy,
			&resultPersistentGathering.Gathering.PolicyArgument,
			&resultPersistentGathering.Gathering.Flags,
			&resultPersistentGathering.Gathering.State,
			&resultPersistentGathering.Gathering.Description,
			&resultPersistentGathering.CommunityType,
			&resultPersistentGathering.Password,
			pqextended.Array(&resultAttribs),
			&resultPersistentGathering.ApplicationBuffer,
			&resultParticipationStartDate,
			&resultParticipationEndDate,
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		resultMatchmakeSessionCount, nexError := GetPersistentGatheringSessionCount(manager, uint32(resultPersistentGathering.ID))
		if nexError != nil {
			common_globals.Logger.Critical(nexError.Error())
			continue
		}

		resultParticipationCount, nexError := GetPersistentGatheringParticipationCount(manager, uint32(resultPersistentGathering.ID), uint64(sourcePID))
		if nexError != nil {
			common_globals.Logger.Critical(nexError.Error())
			continue
		}

		attributesSlice := make([]types.UInt32, len(resultAttribs))
		for i, value := range resultAttribs {
			attributesSlice[i] = types.NewUInt32(value)
		}
		resultPersistentGathering.Attribs = attributesSlice

		resultPersistentGathering.ParticipationStartDate = resultPersistentGathering.ParticipationStartDate.FromTimestamp(resultParticipationStartDate)
		resultPersistentGathering.ParticipationEndDate = resultPersistentGathering.ParticipationEndDate.FromTimestamp(resultParticipationEndDate)
		resultPersistentGathering.MatchmakeSessionCount = types.NewUInt32(resultMatchmakeSessionCount)
		resultPersistentGathering.ParticipationCount = types.NewUInt32(resultParticipationCount)

		persistentGatherings = append(persistentGatherings, resultPersistentGathering)
	}

	rows.Close()

	return persistentGatherings, nil
}
