package ranking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/v2/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/types"
)

func (commonProtocol *CommonProtocol) getRanking(err error, packet nex.PacketInterface, callID uint32, rankingMode *types.PrimitiveU8, category *types.PrimitiveU32, orderParam *ranking_types.RankingOrderParam, uniqueID *types.PrimitiveU64, principalID *types.PID) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if commonProtocol.GetNearbyRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetNearbyRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if commonProtocol.GetFriendsRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetFriendsRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if commonProtocol.GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if commonProtocol.GetOwnRankingByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetOwnRankingByCategoryAndRankingOrderParam!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Ranking.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()
	callerPid := principalID
	// * PUYOPUYOTETRIS seems to do this sometimes
	if callerPid.Value() == 0 {
		callerPid = connection.PID()
	}

	var rankDataList *types.List[*ranking_types.RankingRankData]
	var totalCount uint32

	if rankingMode.Value == 0 {
		rankDataList, totalCount, err = commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam(category, orderParam)
	} else if rankingMode.Value == 1 {
		rankDataList, totalCount, err = commonProtocol.GetNearbyRankingsAndCountByCategoryAndRankingOrderParam(callerPid, category, orderParam)
	} else if rankingMode.Value == 2 {
		rankDataList, totalCount, err = commonProtocol.GetFriendsRankingsAndCountByCategoryAndRankingOrderParam(callerPid, category, orderParam)
	} else if rankingMode.Value == 3 {
		rankDataList, totalCount, err = commonProtocol.GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam(callerPid, category, orderParam)
	} else { // rankingMode.Value == 4
		rankDataList, totalCount, err = commonProtocol.GetOwnRankingByCategoryAndRankingOrderParam(callerPid, category, orderParam)
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Ranking.Unknown, "change_error")
	}

	if totalCount == 0 || rankDataList.Length() == 0 {
		return nil, nex.NewError(nex.ResultCodes.Ranking.NotFound, "change_error")
	}

	pResult := ranking_types.NewRankingResult()

	pResult.RankDataList = rankDataList
	pResult.TotalCount = types.NewPrimitiveU32(totalCount)
	pResult.SinceTime = types.NewDateTime(0x1F40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pResult.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetRanking
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetRanking != nil {
		go commonProtocol.OnAfterGetRanking(packet, rankingMode, category, orderParam, uniqueID, principalID)
	}

	return rmcResponse, nil
}
