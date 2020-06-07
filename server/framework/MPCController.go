package framework

import (
	"github.com/google/uuid"
	"math/big"
)

// State ...
type State struct {
	Secret  Share
	Network NetworkStates
}

type NetworkStates int

const (
	VOTING NetworkStates = iota
	RESULT
)

func NewState(currentSecret *Share, point int64) *State {
	if currentSecret == nil {
		return &State{
			Secret:  NewSecret(point, big.NewInt(0), uuid.UUID{}),
			Network: 0,
		}
	}
	return &State{Secret: *currentSecret, Network: 0}
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
	SecretGen(i int) []Share
	DecodeSecret(i int) Share
	Distribute(secret Share)
	VerifySecret(verificationSecrets []Share) bool // Verify via x(1-x) that the secret is valid (is in {0,1} and other stuff for active)

	// Arithmetics Operations on the currentSecret, found in State.
	Add(s Share)
	Multiply_f(a Share, b Share) Share
	Multiply_const(a Share, i int) Share
	Comparison(a Share, b Share) Share
}
