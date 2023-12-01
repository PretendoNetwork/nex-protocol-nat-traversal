package ranking

import (
	"strings"

	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_mario_kart_8 "github.com/PretendoNetwork/nex-protocols-go/ranking/mario-kart-8"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonRankingProtocol *CommonRankingProtocol

type CommonRankingProtocol struct {
	server             *nex.PRUDPServer
	DefaultProtocol    *ranking.Protocol
	MarioKart8Protocol *ranking_mario_kart_8.Protocol

	GetCommonData                                     func(unique_id uint64) ([]byte, error)
	UploadCommonData                                  func(pid uint32, uniqueID uint64, commonData []byte) error
	InsertRankingByPIDAndRankingScoreData             func(pid uint32, rankingScoreData *ranking_types.RankingScoreData, uniqueID uint64) error
	GetRankingsAndCountByCategoryAndRankingOrderParam func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) ([]*ranking_types.RankingRankData, uint32, error)
}

func initDefault(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.DefaultProtocol = ranking.NewProtocol(c.server)
	c.DefaultProtocol.GetCachedTopXRanking = getCachedTopXRanking
	c.DefaultProtocol.GetCachedTopXRankings = getCachedTopXRankings
	c.DefaultProtocol.GetCommonData = getCommonData
	c.DefaultProtocol.GetRanking = getRanking
	c.DefaultProtocol.UploadCommonData = uploadCommonData
	c.DefaultProtocol.UploadScore = uploadScore
}

func initMarioKart8(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.MarioKart8Protocol = ranking_mario_kart_8.NewProtocol(c.server)
	c.MarioKart8Protocol.GetCachedTopXRanking = getCachedTopXRanking
	c.MarioKart8Protocol.GetCachedTopXRankings = getCachedTopXRankings
	c.MarioKart8Protocol.GetCommonData = getCommonData
	c.MarioKart8Protocol.GetRanking = getRanking
	c.MarioKart8Protocol.UploadCommonData = uploadCommonData
	c.MarioKart8Protocol.UploadScore = uploadScore
}

// NewCommonRankingProtocol returns a new CommonRankingProtocol
func NewCommonRankingProtocol(server *nex.PRUDPServer) *CommonRankingProtocol {
	commonRankingProtocol = &CommonRankingProtocol{server: server}

	patch := server.MatchMakingProtocolVersion().GameSpecificPatch

	if strings.EqualFold(patch, "AMKJ") {
		common_globals.Logger.Info("Using Mario Kart 8 Ranking protocol")
		initMarioKart8(commonRankingProtocol)
	} else {
		if patch != "" {
			common_globals.Logger.Infof("Ranking version patch %q not recognized", patch)
		}

		common_globals.Logger.Info("Using default Ranking protocol")
		initDefault(commonRankingProtocol)
	}

	return commonRankingProtocol
}
