package src

import (
	f "../framework"
	"github.com/google/uuid"
	"math/big"
	"sync"
)

type chanStruct struct {
	share *f.Share
	proof []big.Int
}

type ActiveMPCController struct {
	ss           f.SecretSharing
	network      f.Network
	arithmetic   f.ArithmeticLogic
	state        f.State
	stateChannel chan chanStruct

	startShare    f.Share
	unVerifiedShares map[uuid.UUID]*f.Share
	verificationSets  map[uuid.UUID][]f.Share
	corruptShareId   map[uuid.UUID]bool
	verifiedShares   map[uuid.UUID]bool
	stateLock         *sync.RWMutex
	shareLock        *sync.RWMutex
	verificationLock  *sync.RWMutex

	logger    Logger
	sessionId uuid.UUID
	serverId  int64
}

func (p *ActiveMPCController) RevokeVote(share f.Share, proof []big.Int) {
	negativeShare, negativeProof := p.arithmetic.Multiply_const(share, big.NewInt(-1), proof)
	p.stateChannel <- chanStruct{
		share: &negativeShare,
		proof: negativeProof,
	}
}

func NewActiveMPCController(log Logger) *ActiveMPCController {
	network := new(NetworkHTTPS)

	// Starts the network, either by setup or joining already created network.
	parties, startShare, startProof, _, PrimeP, PrimeQ, sessionId, serverId := network.StartNetwork(log)
	ss := NewActiveSecretSharing(parties, PrimeP.String(), PrimeQ.String())
	mpcController := &ActiveMPCController{
		ss,
		network,
		NewArithmetic(PrimeP,PrimeQ),
		*f.NewState(&startShare, startProof, startShare.Point),
		make(chan chanStruct),

		startShare,
		make(map[uuid.UUID]*f.Share),
		make(map[uuid.UUID][]f.Share),
		make(map[uuid.UUID]bool),
		make(map[uuid.UUID]bool),

		new(sync.RWMutex),
		new(sync.RWMutex),
		new(sync.RWMutex),
		log,
		sessionId,
		serverId,
	}
	network.AddObserver(mpcController)
	go mpcController.addShare()
	return mpcController
}

// Getters
// TODO: Discuss; do we need getters for these submodules? Are they supposed to be private?

func (p *ActiveMPCController) GetSecretSharing() *f.SecretSharing {
	return &p.ss
}

func (p *ActiveMPCController) GetNetwork() *f.Network {
	return &p.network
}

func (p *ActiveMPCController) GetArithmetic() *f.ArithmeticLogic {
	return &p.arithmetic
}

func (p *ActiveMPCController) GetState() *f.State {
	return &p.state
}

// Methods
func (p *ActiveMPCController) DecodeShare(i int) f.Share {
	panic("implement me")
}

func (p *ActiveMPCController) Distribute(share f.Share) {
	p.network.Flood(share)
}

func (p *ActiveMPCController) VerifyShare(verificationSet []f.Share) bool {
	// For passive we only verify that the vote is 0 or 1.
	// verified: x*(x-1) = 0

	if len(verificationSet) <= (p.ss.GetT() + 1) {
		panic("Verification set was not large enough (<=2T+1)")
	}
	x := p.ss.Reconstruct(verificationSet, 2).Int64()
	Debug(2,"\t Reconstruction was: %v \n",x)
	return x == 0
}

func (p *ActiveMPCController) Add(s f.Share, sProof []big.Int) error {
	//The uncommented part should be unnecessary
	// Update currentShare
	//f !s.Verified {
	//	return errors.New("Unverified share will not be added")
	//}
	p.state.Share, p.state.Proof = p.arithmetic.Add(p.state.Share, s, p.state.Proof, sProof)
	return nil
}

func (p *ActiveMPCController) Multiply_f(a f.Share, b f.Share) f.Share {
	panic("implement me")
}

func (p *ActiveMPCController) Multiply_const(a f.Share, i int) f.Share {
	panic("implement me")
}

func (p *ActiveMPCController) Comparison(a f.Share, b f.Share) f.Share {
	panic("implement me")
}

// Observer methods

func (p *ActiveMPCController) ChangedNetworkState(state f.NetworkStates) {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()
	p.state.Network = state
}

func (p *ActiveMPCController) GetCurrentVote() f.Share {
	Debug(1, "\n\n\n\n-- sending vote %v \n\n\n\nProof: %v\n", p.state.Share, p.state.Proof)
	return p.state.Share
}

func (p *ActiveMPCController) NewShareArrived(share f.Share, proof []big.Int) {
	Debug(2, "Got vote with Id %v for", share.Id)
	//p.stateLock.RLock()
	//defer p.stateLock.RUnlock()

	if p.state.Network == f.VOTING {
		p.shareLock.Lock()
		exist := p.verifiedShares[share.Id]


		if !exist {
			p.stateChannel <- chanStruct{
				share: &share,
				proof: proof,
			}
			// Check for already arrived verification set
		}
		// Add share to
		p.shareLock.Unlock()
	} else if p.state.Network == f.RESULT {

	}
}

func (p *ActiveMPCController) addShare() {
	for p.state.Network != f.RESULT {
		sec := <- p.stateChannel

		//p.stateLock.Lock()
		p.state.Share, p.state.Proof = p.arithmetic.Add(*sec.share, p.state.Share, sec.proof, p.state.Proof)
		//p.stateLock.Unlock()
		Debug(1, "Added share to state from uuid: %v\n", sec.share.Id)
	}
}
