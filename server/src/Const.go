package src

type ApiEndpoints string

const (
	JoinNetwork              ApiEndpoints = "/JoinNetwork"
	NewParticipantInNetwork  ApiEndpoints = "/NewParticipantInNetwork"
	SignalStartPhaseTwo      ApiEndpoints = "/SignalStartPhaseTwo"
	FloodShare               ApiEndpoints = "/FloodShare"
	FloodVerificationShare   ApiEndpoints = "/FloodVerificationShare"
	GetAllParticipants       ApiEndpoints = "/GetAllParticipants"
	GetPublicKey             ApiEndpoints = "/GetPublicKey"
	ReceiveVerificationShare ApiEndpoints = "/ReceiveVerificationShare"
	SendShare                ApiEndpoints = "/SendShare"
	GetPublicKeyAsString     ApiEndpoints = "/GetPublicKeyAsString"
	GetPrime                 ApiEndpoints = "/GetPrime"
	ClientJoinNetwork        ApiEndpoints = "/ClientJoinNetwork"
	GetCurrentShare          ApiEndpoints = "/GetCurrentShare"
	ReceiveServerShare       ApiEndpoints = "NetworkHTTPS.ReceiveServerShare"
	GrantSignature           ApiEndpoints = "NetworkHTTPS.GrantSignature"
	ChallengeVote            ApiEndpoints = "NetworkHTTPS.ChallengeVote"
)

type RPCEndpoints string

const (
	ReceivedFloodShareRPC             RPCEndpoints = "NetworkHTTPS.ReceivedFloodShareRPC"
	ReceivedVerificationFloodShareRPC RPCEndpoints = "NetworkHTTPS.ReceivedVerificationFloodShareRPC"
	JoinNetworkRPC                    RPCEndpoints = "NetworkHTTPS.JoinNetworkRPC"
	FloodNewParticipantInNetworkRPC   RPCEndpoints = "NetworkHTTPS.FloodNewParticipantInNetworkRPC"
	Handshake                         RPCEndpoints = "NetworkHTTPS.Handshake"
)

type RequestType string

const (
	GET  RequestType = "GET"
	POST RequestType = "POST"
)

const (
	//certificate_path = "/home/frederik/Desktop/bachelor/go/certificates/"
	ProtocolType = "Final Active"
	MasterIp   = "127.0.0.1"
	MasterPort = "8080"
	PrimeP     = 23
	PrimeQ     = 11
	PrimeP2048 = "31509752883061552834249330521243574505037975346723312991910583824963156798895017299283175208689867651965052129264656502009685245896535004626090261442021060572448035998108052056982854333191745607826615161867988096954791492071845377775480914226293400841859888752656710887510747829426091300971857999543081855388715636524759809034522668781722480529385521759273935019559657089130833484964143760434747499211567204031714978942600664012620378483702934893228421900372434911366411609364970535815602296811292728125276347604060401967036809805695641948757398945537922788111408229811453312669592469294509592852964252394560655355743"
	PrimeQ2048 = "15754876441530776417124665260621787252518987673361656495955291912481578399447508649641587604344933825982526064632328251004842622948267502313045130721010530286224017999054026028491427166595872803913307580933994048477395746035922688887740457113146700420929944376328355443755373914713045650485928999771540927694357818262379904517261334390861240264692760879636967509779828544565416742482071880217373749605783602015857489471300332006310189241851467446614210950186217455683205804682485267907801148405646364062638173802030200983518404902847820974378699472768961394055704114905726656334796234647254796426482126197280327677871"
	KeySize    = 2048
)
