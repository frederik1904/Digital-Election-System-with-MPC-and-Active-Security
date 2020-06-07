package framework

import (
	"github.com/google/uuid"
	"math/big"
)

// Share Structure for internal secret on server side.
type Share struct {
	PointValue *big.Int
	Point      int64
	Id         uuid.UUID
	Verified   bool
	// authentication here?
}

func (s Share) TransformForNetwork() MarshalFriendlyShare {
	return MarshalFriendlyShare{
		PointValue: s.PointValue.String(),
		Point:      s.Point,
		Id:         s.Id,
		Verified:   s.Verified,
	}
}

type MarshalFriendlyShare struct {
	PointValue string
	Point int64
	Id uuid.UUID
	Verified bool
}

func (m MarshalFriendlyShare) TransformToShare() Share {
	PV, _ := big.NewInt(0).SetString(m.PointValue, 10)
	return Share{
		PointValue: PV,
		Point:      m.Point,
		Id:         m.Id,
		Verified:   m.Verified,
	}
}

func NewSecret(point int64, value *big.Int, id uuid.UUID) Share {
	s := new(Share)
	s.PointValue = value
	s.Point = point
	s.Id = id
	return *s
}

// SecretSharing interface
type SecretSharing interface {
	GetT() int
	SecretGen(secret *big.Int) []Share
	DecodeSecret(sec int) Share
	Reconstruct(secrets []Share, tMult ...int) *big.Int
}
