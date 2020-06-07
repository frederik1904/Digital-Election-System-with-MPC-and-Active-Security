package framework

import (
	"fmt"
	"github.com/google/uuid"
	"math/big"
)

type Share struct {
	S         big.Int
	T         big.Int
	Point     int64
	Id        uuid.UUID
}


type MarshalFriendlyShare struct {
	S, T      string
	Point     int64
	Id        uuid.UUID
}


type ZeroKnowledge struct {
	A0, E0, Z0 big.Int
	A1, E1, Z1 big.Int
}

type ZeroKnowledgeMarshalFriendly struct {
	A0, E0, Z0 string
	A1, E1, Z1 string
}

func (z ZeroKnowledge) TransformForNetwork() ZeroKnowledgeMarshalFriendly {
	return ZeroKnowledgeMarshalFriendly{
		A0: z.A0.String(),
		E0: z.E0.String(),
		Z0: z.Z0.String(),
		A1: z.A1.String(),
		E1: z.E1.String(),
		Z1: z.Z1.String(),
	}
}

func (z ZeroKnowledgeMarshalFriendly) TransformToZeroKnowledge() ZeroKnowledge {
		A0, _ := new(big.Int).SetString(z.A0, 10)
		E0, _ := new(big.Int).SetString(z.E0, 10)
		Z0, _ := new(big.Int).SetString(z.Z0, 10)
		A1, _ := new(big.Int).SetString(z.A1, 10)
		E1, _ := new(big.Int).SetString(z.E1, 10)
		Z1, _ := new(big.Int).SetString(z.Z1, 10)

	return ZeroKnowledge{
		A0: *A0,
		E0: *E0,
		Z0: *Z0,
		A1: *A1,
		E1: *E1,
		Z1: *Z1,
	}
}

func BigIntArrToStrArr(a []big.Int) []string {
	var res []string
	for _, v := range a {
		res = append(res, v.String())
	}
	return res
}

func StringArrToBigIntArr(a []string) []big.Int {
	var res []big.Int
	for _, v := range a {
		tmp, _ := new(big.Int).SetString(v, 10)
		res = append(res, *tmp)
	}
	return res
}

func (s Share) TransformForNetwork() MarshalFriendlyShare {
	return MarshalFriendlyShare{
		S:         s.S.String(),
		T:         s.T.String(),
		Point:     s.Point,
		Id:        s.Id,
	}
}

func (m MarshalFriendlyShare) TransformToShare() Share {
	S, _ := big.NewInt(0).SetString(m.S, 10)
	T, _ := big.NewInt(0).SetString(m.T, 10)
	return Share{
		S:         *S,
		T:         *T,
		Point:     m.Point,
		Id:        m.Id,
	}
}

func (zk *ZeroKnowledge) String() string {
	return fmt.Sprintf("P0: (%v,%v,%v) \nP1: (%v,%v,%v) \nK:%v", &zk.A0, &zk.E0, &zk.Z0, &zk.A1, &zk.E1, &zk.Z1)
}

func NewShare(point int64, id uuid.UUID, valueS, valueT *big.Int) Share {
	s := new(Share)
	s.Point = point
	s.Id = id
	// Values
	s.S = *valueS
	s.T = *valueT
	return *s
}

// SecretSharing interface
type SecretSharing interface {
	GetT() int
	SecretGen(secret *big.Int) ([]Share, []big.Int, ZeroKnowledge)
	Reconstruct(shares []Share, tMult ...int) *big.Int
	VerifyShare(share Share, proof []big.Int, knowledge ...*ZeroKnowledge) bool
}
