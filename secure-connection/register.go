package secureconnection

import (
	"strconv"
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func register(err error, client *nex.Client, callID uint32, stationUrls []*nex.StationURL) {
	server := commonSecureConnectionProtocol.server
	missingHandler := false
	if commonSecureConnectionProtocol.addConnectionHandler == nil {
		logger.Warning("Missing AddConnectionHandler!")
		missingHandler = true
	}
	if commonSecureConnectionProtocol.updateConnectionHandler == nil {
		logger.Warning("Missing UpdateConnectionHandler!")
		missingHandler = true
	}
	if commonSecureConnectionProtocol.doesConnectionExistHandler == nil {
		logger.Warning("Missing DoesConnectionExistHandler!")
		missingHandler = true
	}
	if missingHandler {
		return
	}
	localStation := stationUrls[0]
	pidConnectionID := uint32(server.ConnectionIDCounter().Increment())
	client.SetConnectionID(pidConnectionID)
	//localStation.SetPID(strconv.Itoa(int(client.PID())))
	//localStation.SetRVCID(strconv.Itoa(int(pidConnectionID)))
	//localStation.SetPl("2")
	//localStation.SetNatf("2")
	//localStation.SetNatm("1")
	localStationURL := localStation.EncodeToString()
	client.SetLocalStationUrl(localStationURL)

	address := client.Address().IP.String()
	port := strconv.Itoa(client.Address().Port)
	natf := "0"
	natm := "0"
	type_ := "3"

	localStation.SetAddress(address)
	localStation.SetPort(port)
	localStation.SetNatf(natf)
	localStation.SetNatm(natm)
	localStation.SetType(type_)

	urlPublic := localStation.EncodeToString()
	fmt.Println(urlPublic)

	if !commonSecureConnectionProtocol.doesConnectionExistHandler(pidConnectionID) {
		commonSecureConnectionProtocol.addConnectionHandler(pidConnectionID, []string{localStationURL, urlPublic}, address, port)
	} else {
		commonSecureConnectionProtocol.updateConnectionHandler(pidConnectionID, []string{localStationURL, urlPublic}, address, port)
	}

	retval := nex.NewResultSuccess(nex.Errors.Core.Unknown)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteResult(retval) // Success
	rmcResponseStream.WriteUInt32LE(pidConnectionID)
	rmcResponseStream.WriteString(urlPublic)

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SecureMethodRegister, rmcResponseBody)

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
}
