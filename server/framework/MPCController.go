package framework

import (
	"github.com/google/uuid"
	"math/big"
)

// State ...
type State struct {
	Share  Share
	Proof []big.Int
	Network NetworkStates
}

type NetworkStates int

const (
	VOTING NetworkStates = iota
	RESULT
)

func NewState(currentShare *Share, proof []big.Int, point ...int64) *State {
	if currentShare == nil {
		return &State{
			Share:  Share{
				S:     *big.NewInt(0),
				T:     *big.NewInt(0),
				Point: point[0],
				Id:    uuid.UUID{},
			},
			Network: 0,
			Proof: proof,
		}
	}
	return &State{Share: *currentShare, Network: 0}
}

// MPCController øv bøv
type MPCController interface {
	// Fields
	GetSecretSharing() *SecretSharing
	GetObserver() *NetworkObserver
	GetNetwork() *Network
	GetArithmetic() *ArithmeticLogic
	GetState() *State

	// Secrets
	SecretGen(secret *big.Int) ([]Share, []big.Int, ZeroKnowledge)
	VerifyShare(share Share, proof []big.Int, knowledge ZeroKnowledge) bool

	// Arithmetics Operations on the currentShare, found in State.
	Add(a, aProof []big.Int) (Share, []big.Int)
	Multiply_f(a Share, b Share) Share
	Multiply_const(a Share, i *big.Int, aProof []big.Int) (Share, []big.Int)
	Comparison(a Share, b Share) Share
}
