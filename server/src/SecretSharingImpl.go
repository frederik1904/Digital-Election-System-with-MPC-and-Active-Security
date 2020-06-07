package src

import (
	"../framework"
	"crypto/rand"
	"fmt"
	uuid2 "github.com/google/uuid"
	"math/big"
)

type PassiveSecretSharing struct {
	Parties int
	T       int
	prime   *big.Int
}

func (s *PassiveSecretSharing) GetT() int {
	return s.T
}

func NewSecretSharing(parties int, prime *big.Int) *PassiveSecretSharing {
	var s PassiveSecretSharing
	// Variables here
	s.Parties = parties
	// t < n/2
	s.T = CalculateT(parties)
	s.prime = prime

	return &s
}

func (s *PassiveSecretSharing) SecretGen(secret *big.Int) []framework.Share {
	var coefs []big.Int
	for i := 1; i <= s.T; i++ {
		a, _ := rand.Int(rand.Reader, s.prime)
		coefs = append(coefs, *a)
	}

	var secrets []framework.Share
	uuid := uuid2.New()
	Debug(2, "Coefs: %v\n", coefs)
	for x := 1; x <= s.Parties; x++ {
		secrets = append(secrets, framework.NewSecret(int64(x), s.polynomial(secret, big.NewInt(int64(x)), coefs), uuid))
	}

	return secrets
}

func (s *PassiveSecretSharing) polynomial(secret, x *big.Int, coefs []big.Int) *big.Int {
	point, _ := big.NewInt(0).SetString(secret.String(), 10)
	slave := big.NewInt(0)
	polynomialString := secret.String()
	for i, a := range coefs {
		xpow := slave.Exp(x, big.NewInt(int64(i+1)), nil)
		polynomialString += fmt.Sprintf(" + %sx^%d", a.String(), i+1)
		point.Add(point, xpow.Mul(xpow, &a))
	}
	Debug(3, "polynomial: %s = %s\n", polynomialString, point)
	Debug(3, "polynomial: pointmod: %s\n", slave.Mod(point, s.prime).String())
	return slave.Mod(point, s.prime)
}

func DecodeSecret() {

}

func (s *PassiveSecretSharing) Reconstruct(secrets []framework.Share, tMult ...int) *big.Int {
	if tMult == nil {
		return reconstruct(secrets, s.T, *s.prime)
	} else {
		return reconstruct(secrets, tMult[0]*s.T, *s.prime)
	}
}

func reconstruct(secrets []framework.Share, t int, prime big.Int) *big.Int {
	if len(secrets) <= t {
		panic("Not enough secrets to reconstruct value")
	}

	var points []*big.Int
	var values []*big.Int

	result := new(big.Int)

	for _, v := range secrets {
		points = append(points, big.NewInt(v.Point))
		values = append(values, v.PointValue)
	}

	Debug(2, "Reconstruct: Points=%v\n", points)
	Debug(2, "Reconstruct: Values=%v\n", values)

	for i, v := range points {
		tmp := make([]*big.Int, len(points))
		copy(tmp, points)
		j := v
		m := append(tmp[:i], tmp[i+1:]...)

		Debug(2, "Reconstruct: Lagrange points (m) = %v\n", m)

		delta := Lagrange(j, prime, m)
		Debug(2, "Reconstruct: Lagrange = %s\n", delta.String())
		result.Add(delta.Mul(delta, values[i]), result)
	}

	return result.Mod(result, &prime)
}

func Lagrange(i *big.Int, mod big.Int, m []*big.Int) *big.Int {
	slave := big.NewInt(0)
	num := big.NewInt(1)
	denom := big.NewInt(1)
	for _, j := range m {
		num.Mul(num, slave.Neg(j)) // X is always 0 as we only want the secret.
		denom.Mul(denom, slave.Sub(i, j))
	}

	Debug(3, "Lagrange: num = %s\n", num.String())
	Debug(3, "Lagrange: Denom = %s\n", denom.String())

	// get modded denom
	denom.Exp(slave.ModInverse(denom, &mod), big.NewInt(1), &mod)

	Debug(3, "Lagrange: Denom modded inverse = %s\n", denom.String())

	return slave.Mod(num.Mul(num, denom), &mod)
}

func (s *PassiveSecretSharing) DecodeSecret(sec int) framework.Share {
	panic("implement me")
}
