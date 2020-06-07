package src

import (
	"../framework"
	"fmt"
	"math/big"
)

type SimpleArithmetic struct{
	Prime *big.Int
}

func NewSimpleArithmetic(prime *big.Int) *SimpleArithmetic {
	return &SimpleArithmetic{prime}
}

func (s *SimpleArithmetic) Add(a, b framework.Share) framework.Share {
	validate(a, b)
	fmt.Printf("A: %v\nB: %v\n", a, b)
	tmp := big.NewInt(0).Add(a.PointValue, b.PointValue)
	res := tmp.Mod(tmp, s.Prime)
	return framework.NewSecret(a.Point, res, a.Id)
}

func (s *SimpleArithmetic) Multiply_f(a, b framework.Share) framework.Share {
	validate(a, b)
	return framework.NewSecret(a.Point, big.NewInt(0).Mod(big.NewInt(0).Mul(a.PointValue,b.PointValue),s.Prime), a.Id)
}

func (s *SimpleArithmetic) Multiply_const(a framework.Share, i *big.Int) framework.Share {
	a.PointValue = big.NewInt(0).Mod(big.NewInt(0).Mul(a.PointValue, i), s.Prime)
	return framework.NewSecret(a.Point, a.PointValue, a.Id)
}

func validate(a, b framework.Share) {
	/*
		if a.Point != b.Point {
			panic(fmt.Errorf("Differing input x(n) a(n): %d, b(n): %d ", a.Point, b.Point))
		}
	*/
}

func (s *SimpleArithmetic) Comparison(a framework.Share, b framework.Share) framework.Share {
	panic("implement me")
}
