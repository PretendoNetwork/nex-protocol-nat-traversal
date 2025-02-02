package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// InactivateNotificationDatas marks the notifications of a given user as inactive
func InactivateNotificationDatas(manager *common_globals.MatchmakingManager, sourcePID types.PID) *nex.Error {
	_, err := manager.Database.Exec(`UPDATE matchmaking.notifications SET active=false WHERE source_pid=$1`, sourcePID)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
