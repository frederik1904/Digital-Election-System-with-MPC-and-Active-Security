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
	stateChannel chan f.Share

	minusOneSecret    f.Share
	unVerifiedSecrets map[uuid.UUID]f.Share
	verificationSets  map[uuid.UUID][]f.Share
	corruptSecretId   map[uuid.UUID]bool
	verifiedSecrets   map[uuid.UUID]bool
	test              map[uuid.UUID][]f.Share
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
		make(chan f.Share),

		minusOneSecret,
		make(map[uuid.UUID]f.Share),
		make(map[uuid.UUID][]f.Share),
		make(map[uuid.UUID]bool),
		make(map[uuid.UUID]bool),
		make(map[uuid.UUID][]f.Share),

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

func (p *PassiveMPCController) VerifySecret(verificationSet []f.Share) bool {
	// For passive we only verify that the vote is 0 or 1.
	// verified: x*(x-1) = 0

	if len(verificationSet) <= (p.ss.GetT() + 1) {
		panic("Verification set was not large enough (<=2T+1)")
	}
	return p.ss.Reconstruct(verificationSet, 2).Int64() == 0
}

func (p *PassiveMPCController) CreateVerificationSecret(secret *f.Share) f.Share {
	fmt.Printf("\nCreateVerificationSecret Minus one: %v \n", p.minusOneSecret)
	return p.arithmetic.Multiply_f(*secret, p.arithmetic.Add(*secret, p.minusOneSecret))
}

func (p *PassiveMPCController) Add(s f.Share) error {
	//The uncommented part should be unnecessary
	// Update currentSecret
	//f !s.Verified {
	//	return errors.New("Unverified secret will not be added")
	//}
	p.state.Share = p.arithmetic.Add(p.state.Share, s)
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
	Debug(1, "\n\n\n\n-- sending vote %v \n\n\n\n", p.state.Share)
	return p.state.Share
}

func (p *PassiveMPCController) NewShareArrived(secret f.Share) {
	Debug(2, "Got vote with Id %v for", secret.Id)
	//p.stateLock.RLock()
	//defer p.stateLock.RUnlock()
	p.secretLock.Lock()
	defer p.secretLock.Unlock()
	fmt.Println(secret)

	if p.state.Network == f.VOTING {
		exist := p.verifiedSecrets[secret.Id]
		if !exist {

			p.unVerifiedSecrets[secret.Id] = secret

			// Check for already arrived verification set
			verificationSet, foundSet := p.verificationSets[secret.Id]
			if foundSet {
				print("SET ALREADY EXISTS")
				p.VerificationSecretArrived(secret.Id, append(verificationSet, p.CreateVerificationSecret(&secret)), secret)
			} else {
				Debug(2, "Sending verification flood for %v", secret.Id)
				tmp, _ := new(big.Int).SetString(secret.PointValue.String(), 10)
				verificationSecret := p.CreateVerificationSecret(&f.Share{
					PointValue: tmp,
					Point:      secret.Point,
					Id:         secret.Id,
					Verified:   secret.Verified,
				})
				fmt.Printf("\nVerification share created with value: %v\n", verificationSecret)
				p.network.VerificationFlood(verificationSecret)
				//p.stateChannel <- &secret
			}
		} else if p.state.Network == f.RESULT {

		}
		// Add secret to
	}
}

func (p *PassiveMPCController) VerificationSecretArrived(id uuid.UUID, verificationSet []f.Share, s f.Share) {
	Debug(2, "Got vote with Id %v for verification", id)
	p.secretLock.Lock()
	defer p.secretLock.Unlock()

	switch p.state.Network {
	case f.VOTING:
		p.verificationSets[id] = verificationSet
			if p.VerifySecret(verificationSet) {
				Debug(2, "Verification with Id %v got verified", id)
				p.stateChannel <- s
			} else {
				p.corruptSecretId[id] = true
			}
	case f.RESULT:
		return
	}
}

func (p *PassiveMPCController) addShare() {
	for p.state.Network != f.RESULT {
		sec := <-p.stateChannel

		//p.stateLock.Lock()
		p.state.Share = p.arithmetic.Add(sec, p.state.Share)
		//p.stateLock.Unlock()
		fmt.Printf("Our state is: %v", p.state.Share)

		Debug(2, "Added secret to state %v from uuid: %v", p.state.Share, sec.Id)
	}
}
