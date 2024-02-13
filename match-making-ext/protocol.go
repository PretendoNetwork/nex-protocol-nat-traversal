package match_making_ext

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
)

type CommonProtocol struct {
	endpoint                nex.EndpointInterface
	protocol                match_making_ext.Interface
	OnAfterEndParticipation func(acket nex.PacketInterface, idGathering *types.PrimitiveU32, strMessage *types.String)
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making_ext.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerEndParticipation(commonProtocol.endParticipation)

	return commonProtocol
}
