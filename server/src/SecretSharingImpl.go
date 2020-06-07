package src

import (
	f "../framework"
	"crypto/rand"
	"fmt"
	uuid "github.com/google/uuid"
	"github.com/pkg/errors"
	sha3 "golang.org/x/crypto/sha3"
	b "math/big"
)

type ActiveSecretSharing struct {
	Parties int
	T       int
	PrimeP  *b.Int
	PrimeQ  *b.Int
	TBit    int64 // Fixed so that 2^t < q
	g       *b.Int
	h       *b.Int
}

func NewActiveSecretSharing(parties int, primes ...string) *ActiveSecretSharing {
	var s ActiveSecretSharing
	// Variables here
	s.Parties = parties
	s.T = CalculateT(parties)
	if len(primes) != 0 {
		p, _ := new(b.Int).SetString(primes[0], 10)
		q, _ := new(b.Int).SetString(primes[1], 10)
		s.SetPrimes(p, q)
	} else {
		s.SetPrimes(b.NewInt(int64(PrimeP)), b.NewInt(int64(PrimeQ)))
	}
	return &s
}

// SetPrimes sets the primes and Tbit of ActiveSecretSharing.
func (a *ActiveSecretSharing) SetPrimes(p, q *b.Int) {
	a.PrimeP = p
	a.PrimeQ = q
	a.TBit = int64(a.PrimeQ.BitLen() - 1)
	a.SetCommitmentValues()
}

func (a *ActiveSecretSharing) SetCommitmentValues() (*b.Int, *b.Int) {
	var g, h *b.Int
	i := b.NewInt(2)
	for {
		if g == nil {
			g = a._setCommitmentCheck(i)
			i.Add(i, b.NewInt(1))
		} else if h == nil {
			h = a._setCommitmentCheck(i)
			i.Add(i, b.NewInt(1))
		} else {
			break
		}
	}
	a.g, a.h = g, h
	Debug(2, "SetCommitments: g: %v, h: %v\n", g, h)
	return a.g, a.h
}

func (a *ActiveSecretSharing) GetT() int {
	return a.T
}

func (a *ActiveSecretSharing) SecretGen(secret *b.Int) ([]f.Share, []b.Int, f.ZeroKnowledge) {
	if a.g == nil || a.h == nil {
		a.SetCommitmentValues()
	}
	polyF := []b.Int{*secret}
	g0, _ := rand.Int(rand.Reader, a.PrimeQ)
	polyG := []b.Int{*g0}
	proof := []b.Int{*a.BC(secret, g0)}
	for i := 1; i <= a.T; i++ {
		fi, _ := rand.Int(rand.Reader, a.PrimeQ)
		polyF = append(polyF, *fi)
		gi, _ := rand.Int(rand.Reader, a.PrimeQ)
		polyG = append(polyG, *gi)
		proof = append(proof, *a.BC(fi, gi))
	}
	var zk *f.ZeroKnowledge = nil
	var shares []f.Share
	voteId := uuid.New()
	Debug(2, "CoefsF: %v\n", polyF)
	Debug(2, "CoefsG: %v\n", polyG)
	zk = a.createZeroKnowledge(g0, secret.Int64(), &proof[0], voteId)
	for x := int64(1); x <= int64(a.Parties); x++ {
		s := a.polynomial(b.NewInt(x), polyF)
		t := a.polynomial(b.NewInt(x), polyG)
		share := f.NewShare(x, voteId, s, t)
		shares = append(shares, share)
	}
	return shares, proof, *zk
}

func (as *ActiveSecretSharing) createZeroKnowledge(g0 *b.Int, secret int64, e0 *b.Int, voteId uuid.UUID) *f.ZeroKnowledge {
	// Simulate for 1-secret
	var asim, esim, zsim *b.Int

	// Correct:
	r, _ := rand.Int(rand.Reader, as.PrimeQ)
	a := new(b.Int).Exp(as.h, r, as.PrimeP)
	var s *b.Int
	if secret == 0 {
		asim, esim, zsim = as.simulateZeroKnowledge(e0, 1)
		s = as.RandomOracle(a, asim, voteId)
	} else if secret == 1 || DEBUG > 0 {
		asim, esim, zsim = as.simulateZeroKnowledge(e0, 0)
		s = as.RandomOracle(asim, a, voteId)
	} else {
		panic("secret must be 0 or 1")
	}
	e := new(b.Int).Xor(s, esim)
	z := new(b.Int).Mod(new(b.Int).Add(r, new(b.Int).Mul(e, g0)), as.PrimeQ)
	var zk f.ZeroKnowledge
	if secret == 0 {
		 zk = f.ZeroKnowledge{
			A0: *a, E0: *e, Z0: *z,
			A1: *asim, E1: *esim, Z1: *zsim,
		}
	} else if secret == 1 || DEBUG > 0 {
		zk = f.ZeroKnowledge{
			A0: *asim, E0: *esim, Z0: *zsim,
			A1: *a, E1: *e, Z1: *z,
		}
	} else {
		panic("secret must be 0 or 1")
	}
	return &zk
}

func (as *ActiveSecretSharing) RandomOracle(a1, a2 *b.Int, voteId uuid.UUID) *b.Int {
	s := make([]byte, as.TBit)
	sha3.ShakeSum256(s, append(append(a1.Bytes(), a2.Bytes()...), voteId.String()...))
	return new(b.Int).Mod(new(b.Int).SetBytes(s), new(b.Int).Exp(b.NewInt(2), b.NewInt(as.TBit), as.PrimeQ))
}

// simulateZeroKnowledge simulates and accepting a,e,z with e0 for secret = {1,0}
func (as *ActiveSecretSharing) simulateZeroKnowledge(proof0 *b.Int, secret int64) (*b.Int, *b.Int, *b.Int) {
	z, _ := rand.Int(rand.Reader, as.PrimeP)
	if z.Cmp(b.NewInt(0)) == 0 || z.Cmp(b.NewInt(1)) == 0 {
		return as.simulateZeroKnowledge(proof0, secret) // retry if z == 0
	}
	e, _ := rand.Int(rand.Reader, new(b.Int).Exp(b.NewInt(2), b.NewInt(as.TBit), as.PrimeQ))
	gInv := new(b.Int).ModInverse(as.g, as.PrimeP)
	hz := new(b.Int).Exp(as.h, z, as.PrimeP)
	var tmp *b.Int
	if secret == 0 {
		Debug(2,"Simulated secret 0")
		tmp = new(b.Int).Exp(proof0, e, as.PrimeP)
		tmp.ModInverse(tmp, as.PrimeP)
	} else if secret == 1 {
		Debug(2,"Simulated secret 1")
		tmp = new(b.Int).Exp(new(b.Int).Mul(proof0, gInv), e, as.PrimeP)
		tmp.ModInverse(tmp, as.PrimeP)
	} else {
		panic("secret must be 0 or 1")
	}
	a := new(b.Int).Mod(new(b.Int).Mul(hz, tmp), as.PrimeP)
	return a, e, z
}

func (a *ActiveSecretSharing) DecodeShare(share int) f.Share {
	panic("implement me")
}

func (a *ActiveSecretSharing) Reconstruct(shares []f.Share, tMult ...int) *b.Int {
	if tMult == nil {
		return reconstruct(shares, a.T, *a.PrimeQ)
	} else {
		return reconstruct(shares, tMult[0]*a.T, *a.PrimeQ)
	}
}

func (a *ActiveSecretSharing) Verify01(zk *f.ZeroKnowledge, e0 *b.Int, voteId uuid.UUID) error {

	errorMsg := ""

	// xor check
	s := a.RandomOracle(&zk.A0, &zk.A1, voteId)
	xor := new(b.Int).Xor(&zk.E1, &zk.E0)

	Debug(2,"s=%v, xor=%v, e0=%v, e1=%v\n", s, xor, &zk.E0, &zk.E1)

	if xor.Cmp(s) != 0 {
		errorMsg += fmt.Sprint("Verification error: XOR s != e0 ^ e1\n")
	}

	// Accepting
	gInv := new(b.Int).ModInverse(a.g, a.PrimeP)
	hz0 := new(b.Int).Exp(a.h, &zk.Z0, a.PrimeP)
	ake0 := new(b.Int).Mod(new(b.Int).Mul(&zk.A0, new(b.Int).Exp(e0, &zk.E0, a.PrimeP)), a.PrimeP)
	hz1 := new(b.Int).Exp(a.h, &zk.Z1, a.PrimeP)
	ake1 := new(b.Int).Mod(new(b.Int).Mul(&zk.A1, new(b.Int).Exp(new(b.Int).Mul(e0, gInv), &zk.E1, a.PrimeP)), a.PrimeP)

	accept0 := hz0.Cmp(ake0) == 0
	Debug(2,"0: accepting: %v, values: %v, %v, ", accept0, hz0, ake0)
	accept1 := hz1.Cmp(ake1) == 0
	Debug(2,"1: accepting: %v, values: %v, %v, ", accept1, hz1, ake1)

	if !(accept0 && accept1) {
		errorMsg += fmt.Sprintf("Verification error: Non-accepting conversations (0: %v, 1: %v)", accept0, accept1)
	}

	if errorMsg == "" {
		return nil
	}
	return errors.New(errorMsg)

}

func (a *ActiveSecretSharing) VerifyShare(share f.Share, proof []b.Int, knowledge ...*f.ZeroKnowledge) bool {
	bc := a.BC(&share.S, &share.T)
	proofResult := b.NewInt(1)
	for i, value := range proof {
		// Prod_j^
		proofResult.Mul(proofResult, new(b.Int).Exp(&value, new(b.Int).Exp(b.NewInt(share.Point), b.NewInt(int64(i)), nil), a.PrimeP))
	}
	proofResult.Mod(proofResult, a.PrimeP)

	Debug(2, "Verifying share; ID: %s, Point: %v, \n", share.Id, share.Point)
	Debug(2, "\tBC(s,t) = \t%v\n", bc)
	Debug(2, "\tproof = \t%v\n", proof)

	return bc.Cmp(proofResult) == 0 && (len(knowledge) == 0 || a.Verify01(knowledge[0], &proof[0], share.Id) == nil)
}

// Helpers

func (a *ActiveSecretSharing) polynomial(x *b.Int, polynomial []b.Int) *b.Int {
	polynomialString := polynomial[0].String()
	value, err := new(b.Int).SetString(polynomialString, 10)
	if !err {
		panic("polynomial: string conversion failed")
	}
	slave := b.NewInt(0)
	for i, coef := range polynomial[1:] {
		xpow := slave.Exp(x, b.NewInt(int64(i+1)), a.PrimeQ)
		polynomialString += fmt.Sprintf(" + %sx^%d", coef.String(), i+1)
		value.Add(value, xpow.Mul(xpow, &coef))
	}
	Debug(3, "polynomial: %s = %s\n", polynomialString, value)
	Debug(3, "polynomial: pointmod: %s\n", slave.Mod(value, a.PrimeQ).String())
	return slave.Mod(value, a.PrimeQ)
}

// BC the function g^s * h^t mod p
func (a *ActiveSecretSharing) BC(s, t *b.Int) *b.Int {
	gexp := new(b.Int).Exp(a.g, s, a.PrimeP) // g^s
	hexp := new(b.Int).Exp(a.h, t, a.PrimeP) // h^t
	//fmt.Printf("%v * %v mod %v\n", gexp, hexp, a.PrimeP)
	mult := new(b.Int).Mul(gexp, hexp)    // g^s * h^t
	return new(b.Int).Mod(mult, a.PrimeP) // mod p
}

func (a *ActiveSecretSharing) _setCommitmentCheck(i *b.Int) *b.Int {
	if new(b.Int).Exp(i, a.PrimeQ, a.PrimeP).Cmp(b.NewInt(1)) == 0 {
		res , _ := new(b.Int).SetString(i.String(), 10)
		return res
	} else {
		return nil
	}
}

func Lagrange(i *b.Int, mod b.Int, m []*b.Int) *b.Int {
	slave := b.NewInt(0)
	num := b.NewInt(1)
	denom := b.NewInt(1)
	for _, j := range m {
		num.Mul(num, slave.Neg(j)) // X is always 0 as we only want the secret.
		denom.Mul(denom, slave.Sub(i, j))
	}

	Debug(3, "Lagrange: num = %s\n", num.String())
	Debug(3, "Lagrange: Denom = %s\n", denom.String())

	// get modded denom
	denom.Exp(slave.ModInverse(denom, &mod), b.NewInt(1), &mod)

	Debug(3, "Lagrange: Denom modded inverse = %s\n", denom.String())

	return slave.Mod(num.Mul(num, denom), &mod)
}

func reconstruct(shares []f.Share, t int, prime b.Int) *b.Int {
	if len(shares) <= t {
		panic("Not enough secrets to reconstruct value")
	}

	var points []*b.Int
	var values []b.Int

	result := new(b.Int)

	for _, v := range shares {
		points = append(points, b.NewInt(v.Point))
		values = append(values, v.S)
	}

	Debug(2, "Reconstruct: Points=%v\n", points)
	Debug(2, "Reconstruct: Values=%v\n", values)

	for i, v := range points {
		tmp := make([]*b.Int, len(points))
		copy(tmp, points)
		j := v
		m := append(tmp[:i], tmp[i+1:]...)

		Debug(2, "Reconstruct: Lagrange points (m) = %v\n", m)

		delta := Lagrange(j, prime, m)
		Debug(2, "Reconstruct: Lagrange = %s\n", delta.String())
		result.Add(delta.Mul(delta, &values[i]), result)
	}

	return result.Mod(result, &prime)
}
