package common_globals

import (
	"database/sql"
	"sync"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// MatchmakingManager manages a matchmaking instance
type MatchmakingManager struct {
	Database                 *sql.DB
	Endpoint                 *nex.PRUDPEndPoint
	Mutex                    *sync.RWMutex
	GetUserFriendPIDs        func(pid uint32) []uint32
	GetDetailedGatheringByID func(manager *MatchmakingManager, sourcePID uint64, gatheringID uint32) (types.RVType, string, *nex.Error)
}

// NewMatchmakingManager returns a new MatchmakingManager
func NewMatchmakingManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *MatchmakingManager {
	return &MatchmakingManager{
		Endpoint: endpoint,
		Database: db,
		Mutex:    &sync.RWMutex{},
	}
}
