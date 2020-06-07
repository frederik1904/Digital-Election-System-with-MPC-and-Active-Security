package src

import (
	f "../framework"
	"fmt"
	"math/big"
)

type ActiveArithmetic struct{
	PrimeP  *big.Int
	PrimeQ  *big.Int
}

func NewArithmetic(p, q *big.Int) *ActiveArithmetic {
	return &ActiveArithmetic{
		p,
		q,
	}
}

func (s *ActiveArithmetic) Add(a, b f.Share, aProof, bProof []big.Int) (f.Share, []big.Int) {
	validate(a, b)
	if bProof == nil {
		return a, aProof
	}

	_s := new(big.Int).Mod(new(big.Int).Add(&a.S, &b.S), s.PrimeQ)
	_t := new(big.Int).Mod(new(big.Int).Add(&a.T, &b.T), s.PrimeQ)
	var proof []big.Int

	for i, sp := range aProof {
		mul := new(big.Int).Mul(&sp, &bProof[i])
		mod := new(big.Int).Mod(mul, s.PrimeP)
		proof = append(proof, *mod)
	}


	return f.NewShare(a.Point, a.Id, _s, _t), proof
}

func (s *ActiveArithmetic) Multiply_f(a, b f.Share) f.Share {
	validate(a, b)
	panic("Not implemented")
}

func (s *ActiveArithmetic) Multiply_const(share f.Share, a *big.Int, aProof []big.Int) (f.Share, []big.Int) {
	_s := new(big.Int).Mod(new(big.Int).Mul(a, &share.S), s.PrimeQ)
	_t := new(big.Int).Mod(new(big.Int).Mul(a, &share.T), s.PrimeQ)
	proof := []big.Int{}
	for _, p := range aProof {
		proof = append(proof,*new(big.Int).Exp(&p, a, s.PrimeP))
	}
	return f.NewShare(share.Point, share.Id, _s, _t), proof
}

func validate(a, b f.Share) {
	if a.Point != b.Point {
		panic(fmt.Errorf("Different share points x(n); a(n): %d, b(n): %d ", a.Point, b.Point))
	}
}

func (s *ActiveArithmetic) Comparison(a f.Share, b f.Share) f.Share {
	panic("implement me")
}