package tracking

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// LogParticipateCommunity logs a persistent gathering participation event on the given database
func LogParticipateCommunity(db *sql.DB, sourcePID types.PID, communityGID uint32, gatheringID uint32, participationCount uint32) *nex.Error {
	eventTime := time.Now().UTC()

	_, err := db.Exec(`INSERT INTO tracking.participate_community (
		date,
		source_pid,
		community_gid,
		gathering_id,
		participation_count
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5
	)`, eventTime, uint64(sourcePID), communityGID, gatheringID, participationCount)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
