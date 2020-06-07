package framework

import (
	"github.com/google/uuid"
	"math/big"
)

// https://godoc.org/github.com/ethereum/go-ethereum/p2p
// https://github.com/libp2p/go-libp2p

// NetworkObserver interface
type NetworkObserver interface {
	ChangedNetworkState(state NetworkStates)
	NewShareArrived(secret Share)
	VerificationSecretArrived(id uuid.UUID, verificationSet []Share, s Share)
	GetCurrentVote() Share
}

// Network interface
type Network interface {
	StartNetwork(log interface{}) (int, Share, *big.Int, uuid.UUID, int64) // Starts network and waits for all servers to connect. Returns number of parties.
	AddObserver(networkObserver NetworkObserver)
	RemoveObserver(networkObserver NetworkObserver)
	ChangeNetworkState(state NetworkStates)
	Flood(secret Share)
	VerificationFlood(secret Share)
}
