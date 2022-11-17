package nattraversal

import (
	"fmt"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func requestProbeInitiationExt(err error, client *nex.Client, callID uint32, targetList []string, stationToProbe string) {
	rmcResponse := nex.NewRMCResponse(nexproto.NATTraversalProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.NATTraversalMethodRequestProbeInitiationExt, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PrudpVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	rmcMessage := nex.RMCRequest{}
	rmcMessage.SetProtocolID(nexproto.NATTraversalProtocolID)
	rmcMessage.SetCallID(0xffff0000 + callID)
	rmcMessage.SetMethodID(nexproto.NATTraversalMethodInitiateProbe)
	rmcRequestStream := nex.NewStreamOut(server)
	//stationToProbeUrl := nex.NewStationURL(stationToProbe)
	//stationToProbeUrl.SetCID("1337825000")
	//stationToProbe = stationToProbeUrl.EncodeToString() + ";R=1;Rsa=159.203.102.56;Rsp=9999;Ra=159.203.102.56;Rp=9999"
	rmcRequestStream.WriteString(stationToProbe)
	rmcRequestBody := rmcRequestStream.Bytes()
	rmcMessage.SetParameters(rmcRequestBody)
	rmcMessageBytes := rmcMessage.Bytes()

	stationUrlsStrings := GetConnectionUrlsHandler(client.ConnectionID())
	stationUrls := make([]nex.StationURL, len(stationUrlsStrings))

	for i := 0; i < len(stationUrlsStrings); i++ {
		stationUrls[i] = *nex.NewStationURL(stationUrlsStrings[i])
		stationToProbeUrl := *nex.NewStationURL(stationToProbe)
		if stationUrls[i].Type() == "3" && stationToProbeUrl.Type() == "3" {
			ReplaceConnectionUrlHandler(client.ConnectionID(), stationUrlsStrings[i], stationToProbe)
		}else if stationUrls[i].Type() != "3" && stationToProbeUrl.Type() != "3" {
			ReplaceConnectionUrlHandler(client.ConnectionID(), stationUrlsStrings[i], stationToProbe)
		}
	}

	for _, target := range targetList {
		targetUrl := nex.NewStationURL(target)
		fmt.Println("target: " + target)
		fmt.Println("toProbe: " + stationToProbe)
		targetRvcID, _ := strconv.Atoi(targetUrl.RVCID())
		targetClient := server.FindClientFromConnectionID(uint32(targetRvcID))
		if targetClient != nil {
			var messagePacket nex.PacketInterface

			if server.PrudpVersion() == 0 {
				messagePacket, _ = nex.NewPacketV0(targetClient, nil)
				messagePacket.SetVersion(0)
			} else {
				messagePacket, _ = nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
			}

			messagePacket.SetSource(0xA1)
			messagePacket.SetDestination(0xAF)
			messagePacket.SetType(nex.DataPacket)
			messagePacket.SetPayload(rmcMessageBytes)

			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			server.Send(messagePacket)
		} else {
			fmt.Println("not found")
		}
	}
}
