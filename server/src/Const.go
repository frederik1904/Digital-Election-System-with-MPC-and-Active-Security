package src

type ApiEndpoints string

const (
	JoinNetwork               ApiEndpoints = "/JoinNetwork"
	NewParticipantInNetwork   ApiEndpoints = "/NewParticipantInNetwork"
	SignalStartPhaseTwo       ApiEndpoints = "/SignalStartPhaseTwo"
	FloodSecret               ApiEndpoints = "/FloodSecret"
	FloodVerificationSecret   ApiEndpoints = "/FloodVerificationSecret"
	GetAllParticipants        ApiEndpoints = "/GetAllParticipants"
	GetPublicKey              ApiEndpoints = "/GetPublicKey"
	ReceiveVerificationSecret ApiEndpoints = "/ReceiveVerificationSecret"
	SendShare                 ApiEndpoints = "/SendShare"
	GetPublicKeyAsString      ApiEndpoints = "/GetPublicKeyAsString"
	GetPrime                  ApiEndpoints = "/GetPrime"
	ClientJoinNetwork         ApiEndpoints = "/ClientJoinNetwork"
	GetCurrentShare           ApiEndpoints = "/GetCurrentShare"
)

type RPCEndpoints string

const (
	ReceivedFloodShareRPC             RPCEndpoints = "NetworkHTTPS.ReceivedFloodShareRPC"
	ReceivedVerificationFloodShareRPC RPCEndpoints = "NetworkHTTPS.ReceivedVerificationFloodShareRPC"
	JoinNetworkRPC                    RPCEndpoints = "NetworkHTTPS.JoinNetworkRPC"
	FloodNewParticipantInNetworkRPC   RPCEndpoints = "NetworkHTTPS.FloodNewParticipantInNetworkRPC"
	Handshake                         RPCEndpoints = "NetworkHTTPS.Handshake"
)

type RequestType int

const (
	GET  RequestType = 0
	POST RequestType = 1
)

type ShareType int

const (
	ReceiveShare ShareType = 0
	FloodShare ShareType = 1
	VerificationShare ShareType = 2
)

const (
	ProtocolType = "Second Passive"
	MasterIp   = "127.0.0.1"
	MasterPort = "8080"
	Prime = "10067"
)
