package src

import (
	f "../framework"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"sync"
)

type PassiveMPCController struct {
	ss           f.SecretSharing
	network      f.Network
	arithmetic   f.ArithmeticLogic
	state        f.State
	stateChannel chan *f.Share

	minusOneSecret    f.Share
	unVerifiedSecrets map[uuid.UUID]*f.Share
	verificationSets  map[uuid.UUID][]f.Share
	corruptSecretId   map[uuid.UUID]bool
	verifiedSecrets   map[uuid.UUID]bool
	stateLock         *sync.RWMutex
	secretLock        *sync.RWMutex
	verificationLock  *sync.RWMutex

	logger    Logger
	sessionId uuid.UUID
	serverId  int64
}

func NewPassiveMPCController(log Logger) *PassiveMPCController {
	network := new(NetworkHTTPS)

	// Starts the network, either by setup or joining already created network.
	parties, minusOneSecret, prime, sessionId, serverId := network.StartNetwork(log)
	fmt.Printf("\nMINUS ONE SHARE: %v\n", minusOneSecret)
	ss := NewSecretSharing(parties, prime)
	mpcController := &PassiveMPCController{
		ss,
		network,
		NewSimpleArithmetic(prime),
		*f.NewState(nil, serverId),
		make(chan *f.Share),

		minusOneSecret,
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

func (p *PassiveMPCController) GetSecretSharing() *f.SecretSharing {
	return &p.ss
}

func (p *PassiveMPCController) GetNetwork() *f.Network {
	return &p.network
}

func (p *PassiveMPCController) GetArithmetic() *f.ArithmeticLogic {
	return &p.arithmetic
}

func (p *PassiveMPCController) GetState() *f.State {
	return &p.state
}

// Methods

func (p *PassiveMPCController) SecretGen(i int) []f.Share {
	return p.ss.SecretGen(big.NewInt(int64(i)))
}

func (p *PassiveMPCController) DecodeSecret(i int) f.Share {
	panic("implement me")
}

func (p *PassiveMPCController) Distribute(secret f.Share) {
	p.network.Flood(secret)
}

func (p *PassiveMPCController) Add(s f.Share) error {
	//The uncommented part should be unnecessary
	// Update currentSecret
	//f !s.Verified {
	//	return errors.New("Unverified secret will not be added")
	//}
	p.state.Secret = p.arithmetic.Add(p.state.Secret, s)
	return nil
}

func (p *PassiveMPCController) Multiply_f(a f.Share, b f.Share) f.Share {
	panic("implement me")
}

func (p *PassiveMPCController) Multiply_const(a f.Share, i int) f.Share {
	panic("implement me")
}

func (p *PassiveMPCController) Comparison(a f.Share, b f.Share) f.Share {
	panic("implement me")
}

// Observer methods

func (p *PassiveMPCController) ChangedNetworkState(state f.NetworkStates) {
	p.stateLock.Lock()
	defer p.stateLock.Unlock()
	p.state.Network = state
}

func (p *PassiveMPCController) GetCurrentVote() f.Share {
	Debug(2, "\n\n\n\n-- sending vote %v \n\n\n\n", p.state.Secret)
	return p.state.Secret
}

func (p *PassiveMPCController) NewSecretArrived(share f.Share) {
	Debug(2, "Got vote with Id %v for", share.Id)

	if p.state.Network == f.VOTING {
		p.stateChannel <- &share
	}
}

func (p *PassiveMPCController) addShare() {
	for p.state.Network != f.RESULT {
		sec := <- p.stateChannel
		fmt.Printf("Our state is: %v", p.state.Secret)
		//p.stateLock.Lock()
		p.state.Secret = p.arithmetic.Add(*sec, p.state.Secret)
		//p.stateLock.Unlock()

		Debug(2, "Added secret to state %v from uuid: %v", p.state.Secret, sec.Id)
	}
}
