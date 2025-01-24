package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// CreatePersistentGathering creates a new PersistentGathering on the database. No participants are added
func CreatePersistentGathering(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, persistentGathering *match_making_types.PersistentGathering) *nex.Error {
	_, nexError := match_making_database.RegisterGathering(manager, connection.PID(), types.NewPID(0), &persistentGathering.Gathering, "PersistentGathering")
	if nexError != nil {
		return nexError
	}

	attribs := make([]uint32, len(persistentGathering.Attribs))
	for i, value := range persistentGathering.Attribs {
		attribs[i] = uint32(value)
	}

	_, err := manager.Database.Exec(`INSERT INTO matchmaking.persistent_gatherings (
		id,
		community_type,
		password,
		attribs,
		application_buffer,
		participation_start_date,
		participation_end_date
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7
	)`,
		uint32(persistentGathering.Gathering.ID),
		uint32(persistentGathering.CommunityType),
		string(persistentGathering.Password),
		pqextended.Array(attribs),
		[]byte(persistentGathering.ApplicationBuffer),
		persistentGathering.ParticipationStartDate.Standard(),
		persistentGathering.ParticipationEndDate.Standard(),
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
