package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetDetailedGatheringByID returns a Gathering as an RVType by its gathering ID
func GetDetailedGatheringByID(manager *common_globals.MatchmakingManager, sourcePID uint64, gatheringID uint32) (types.RVType, string, *nex.Error) {
	gathering, gatheringType, _, _, nexError := FindGatheringByID(manager, gatheringID)
	if nexError != nil {
		return nil, "", nexError
	}

	if gatheringType != "Gathering" {
		return nil, "", nex.NewError(nex.ResultCodes.Core.Exception, "change_error")
	}

	return gathering, gatheringType, nil
}
