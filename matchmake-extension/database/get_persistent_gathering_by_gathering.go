package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// GetPersistentGatheringByGathering gets a persistent gathering with the given gathering data
func GetPersistentGatheringByGathering(manager *common_globals.MatchmakingManager, gathering match_making_types.Gathering, sourcePID uint64) (match_making_types.PersistentGathering, *nex.Error) {
	resultPersistentGathering := match_making_types.NewPersistentGathering()
	var resultAttribs []uint32
	var resultParticipationStartDate time.Time
	var resultParticipationEndDate time.Time

	err := manager.Database.QueryRow(`SELECT
		community_type,
		password,
		attribs,
		application_buffer,
		participation_start_date,
		participation_end_date
		FROM matchmaking.persistent_gatherings
		WHERE id=$1`,
		gathering.ID,
	).Scan(
		&resultPersistentGathering.CommunityType,
		&resultPersistentGathering.Password,
		pqextended.Array(&resultAttribs),
		&resultPersistentGathering.ApplicationBuffer,
		&resultParticipationStartDate,
		&resultParticipationEndDate,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return match_making_types.NewPersistentGathering(), nex.NewError(nex.ResultCodes.RendezVous.InvalidGID, "change_error")
		} else {
			return match_making_types.NewPersistentGathering(), nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultPersistentGathering.Gathering = gathering

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultPersistentGathering.Attribs = attributesSlice

	resultPersistentGathering.ParticipationStartDate = resultPersistentGathering.ParticipationStartDate.FromTimestamp(resultParticipationStartDate)
	resultPersistentGathering.ParticipationEndDate = resultPersistentGathering.ParticipationEndDate.FromTimestamp(resultParticipationEndDate)

	return resultPersistentGathering, nil
}
