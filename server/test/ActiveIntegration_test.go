package test

import (
	f "../framework"
	s "../src"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	b "math/big"
	"reflect"
	"testing"
)

func TestArithAdd(t *testing.T) {
	a := s.NewActiveSecretSharing(6)
	a.SetPrimes(b.NewInt(23),  b.NewInt(11))
	//a.SetPrimes(s.FrederiksMaxPrimeFactor(64, 8))
	proof1 := []b.Int{*b.NewInt(12), *b.NewInt(2), *b.NewInt(1)}
	share1 := f.Share{
		S:     *b.NewInt(1),
		T:     *b.NewInt(4),
		Point: 1,
		Id:    uuid.UUID{},
	}
	proof2 := []b.Int{*b.NewInt(8), *b.NewInt(18), *b.NewInt(18)}
	share2 := f.Share{
		S:     *b.NewInt(10),
		T:     *b.NewInt(2),
		Point: 1,
		Id:    uuid.UUID{},
	}

	aa := s.NewArithmetic(b.NewInt(23), b.NewInt(11))
	add, addProof := aa.Add(share1, share2, proof1, proof2)
	// s and t
	if add.S.Cmp(b.NewInt(0)) != 0 || add.T.Cmp(b.NewInt(6)) != 0 {
		t.Error(fmt.Sprintf("Incorrect S and T, found: %v, %v, expected: 0, 6", &add.S, &add.T))
	}
	// proof
	if !reflect.DeepEqual(addProof, []b.Int{*b.NewInt(4), *b.NewInt(13), *b.NewInt(18)}) {
		t.Error(fmt.Sprintf("Incorrect proof, found: %v, expected: [4, 13, 18]", addProof))
	}


}

func TestAddReconstruct100(t *testing.T) {
	for i := 1; i < 100; i++ {
		TestAddReconstruct(t)
	}
}

func TestAddReconstruct(t *testing.T) {
	p, q := s.FrederiksMaxPrimeFactor(64, 4)
	s.Debug(2, "Found p: %v, q: %v\n", p, q)
	ss1 := s.NewActiveSecretSharing(6)
	ss1.PrimeP = p
	ss1.PrimeQ = q
	g, h := ss1.SetCommitmentValues()
	s.Debug(2, "Found g: %v, h: %v\n", g, h)

	ss2 := s.NewActiveSecretSharing(6)
	ss2.PrimeP = p
	ss2.PrimeQ = q
	ss2.SetCommitmentValues()


	a, _ := rand.Int(rand.Reader, b.NewInt(10000))
	c, _ := rand.Int(rand.Reader, b.NewInt(10000))
	s.Debug(2, "Adding a: %v, c: %v\n", a, c)

	shares1, proof1, _ := ss1.SecretGen(a)
	shares2, proof2, _ := ss1.SecretGen(c)
	s.Debug(2,"Created shares:\n 1:\n %v\n 2:\n %v\n", s.PrintShares(shares1), s.PrintShares(shares2))

	ar := s.NewArithmetic(p, q)

	var sharesAdd []f.Share
	addProof := proof1
	for i, s := range shares1 {
		var share f.Share
		share, addProof = ar.Add(s, shares2[i], proof1, proof2)
		sharesAdd = append(sharesAdd, share)
	}

	// Verify add
	for _, share := range sharesAdd {
		if !ss1.VerifyShare(share, addProof) || !ss2.VerifyShare(share, addProof) {
			t.Errorf("Verification failed for sharesAdd, try running with DEBUG=2: S: %v, T: %v, proof: %v, g: %v, h: %v", &share.S, &share.T, addProof, g, h)
		}
	}

	// Reconstruct add
	reconstruct := ss1.Reconstruct(sharesAdd)
	sum := new(b.Int).Add(a, c)
	if reconstruct.Cmp(sum) != 0 {
		t.Error(fmt.Sprintf("Reconstruc failed: found %v, expected %v", reconstruct, sum))
	}

}

func TestMultConstReconstruct(t *testing.T) {
	ss1 := s.NewActiveSecretSharing(6)
	ss1.PrimeP = b.NewInt(23)
	ss1.PrimeQ = b.NewInt(11)
	ss1.SetCommitmentValues()

	ss2 := s.NewActiveSecretSharing(6)
	ss2.PrimeP = b.NewInt(23)
	ss2.PrimeQ = b.NewInt(11)
	ss2.SetCommitmentValues()

	shares1, proof1, _ := ss1.SecretGen(b.NewInt(3))

	ar := s.NewArithmetic(b.NewInt(s.PrimeP), b.NewInt(s.PrimeQ))

	var sharesMultConst []f.Share
	var proof []b.Int
	for _, sh := range shares1 {
		var share f.Share
		share, proof = ar.Multiply_const(sh, b.NewInt(3), proof1)
		sharesMultConst = append(sharesMultConst, share)
	}

	// Verify add
	for _, share := range sharesMultConst {
		if !ss1.VerifyShare(share, proof) || !ss2.VerifyShare(share, proof) {
			t.Error("Verification failed for sharesAdd, try running with DEBUG=2")
		}
	}

	// Reconstruct add
	reconstruct := ss1.Reconstruct(sharesMultConst)
	if reconstruct.Cmp(b.NewInt(9)) != 0 {
		t.Error(fmt.Sprintf("Reconstruc failed: found %v, expected 9", reconstruct))
	}

}

func TestZeroKnowledge(t *testing.T) {
	a := s.NewActiveSecretSharing(6)
	a.SetPrimes(s.FrederiksMaxPrimeFactor(512, 8))
	secret := b.NewInt(1)
	share, proof, zk := a.SecretGen(secret)

	share2, iProof, iZk := a.SecretGen(b.NewInt(3))

	if err := a.Verify01(&zk, &proof[0],share[0].Id); err != nil {
		t.Error(err)
	}
	if err := a.Verify01(&iZk, &iProof[0], share2[0].Id); err == nil {
		t.Error("Failed: Should no verify for invalid shares with secret 3")
	}

}

func TestAddSeveral(t *testing.T) {
	p, q := s.FrederiksMaxPrimeFactor(64, 24)
	s.Debug(2, "Found p: %v, q: %v\n", p, q)
	ar := s.NewArithmetic(p, q)

	ss1 := s.NewActiveSecretSharing(6)
	ss1.PrimeP = p
	ss1.PrimeQ = q
	g, h := ss1.SetCommitmentValues()
	s.Debug(2, "Found g: %v, h: %v\n", g, h)

	ss2 := s.NewActiveSecretSharing(6)
	ss2.PrimeP = p
	ss2.PrimeQ = q
	ss2.SetCommitmentValues()

	shares1, proof1, _ := ss1.SecretGen(b.NewInt(0))


	for i := 1; i < 100; i++ {
		shares2, proof2, _ := ss1.SecretGen(b.NewInt(1))


		var sharesAdd []f.Share
		addProof := proof1
		for i, s := range shares1 {
			var share f.Share
			share, addProof = ar.Add(s, shares2[i], proof1, proof2)
			sharesAdd = append(sharesAdd, share)
		}

		// Verify add
		for _, share := range sharesAdd {
			if !ss1.VerifyShare(share, addProof) || !ss2.VerifyShare(share, addProof) {
				t.Errorf("Verification failed for sharesAdd, try running with DEBUG=2: S: %v, T: %v, proof: %v, g: %v, h: %v", &share.S, &share.T, addProof, g, h)
			}
		}

		// Reconstruct add
		reconstruct := ss1.Reconstruct(sharesAdd)
		if reconstruct.Cmp(b.NewInt(int64(i))) != 0 {
			t.Error(fmt.Sprintf("Reconstruc failed: found %v, expected %v", reconstruct, i))
			break
		}
		shares1, proof1 = sharesAdd, addProof
	}
}

func TestSubtractShare(t *testing.T) {
	p, q := s.FrederiksMaxPrimeFactor(64, 24)
	s.Debug(2, "Found p: %v, q: %v\n", p, q)
	ar := s.NewArithmetic(p, q)

	ss1 := s.NewActiveSecretSharing(6)
	ss1.PrimeP = p
	ss1.PrimeQ = q
	g, h := ss1.SetCommitmentValues()
	s.Debug(2, "Found g: %v, h: %v\n", g, h)

	ss2 := s.NewActiveSecretSharing(6)
	ss2.PrimeP = p
	ss2.PrimeQ = q
	ss2.SetCommitmentValues()

	shares1, proof1, _ := ss1.SecretGen(b.NewInt(0))
	shares2, proof2, _ := ss1.SecretGen(b.NewInt(1))

	// First, add together.
	var sharesAdd []f.Share
	var shareAdd f.Share
	var addProof []b.Int
	for i, s := range shares1 {
		shareAdd, addProof = ar.Add(s, shares2[i], proof1, proof2)
		sharesAdd = append(sharesAdd, shareAdd)
	}

	// Next mult const
	var shares2Negative []f.Share
	var share2Negative f.Share
	var negProof []b.Int
	for _, s := range shares2 {
		share2Negative, negProof = ar.Multiply_const(s, b.NewInt(-1), proof2)
		shares2Negative = append(shares2Negative, share2Negative)
	}

	// Add again
	var sharesSub []f.Share
	var shareSub f.Share
	var subProof []b.Int
	for i, s := range sharesAdd {
		shareSub, subProof = ar.Add(s, shares2Negative[i], addProof, negProof)
		sharesSub = append(sharesSub, shareSub)
	}

	// Verify
	for _, share := range sharesSub {
		if !ss1.VerifyShare(share, subProof) || !ss2.VerifyShare(share, subProof) {
			t.Errorf("Verification failed for sharesSub, try running with DEBUG=2: S: %v, T: %v, proof: %v, g: %v, h: %v", &share.S, &share.T, subProof, g, h)
		}
	}

	// Proofs are the same
	if !reflect.DeepEqual(subProof, proof1) {
		t.Errorf("Proofs were not the same: proof1 %v, subproof: %v", proof1, subProof)
	}

	// Reconstruct
	reconstruct := ss1.Reconstruct(sharesSub)
	if reconstruct.Cmp(b.NewInt(int64(0))) != 0 {
		t.Error(fmt.Sprintf("Reconstruct failed: found %v, expected %v", reconstruct, 0))
	}

}
