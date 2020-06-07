package test

import (
	f "../framework"
	s "../src"
	"fmt"
	"github.com/google/uuid"
	b "math/big"
	"reflect"
	"testing"
)

func TestMaxPrimeFactor(t *testing.T) {
	p := b.NewInt(23)
	q := b.NewInt(11)

	found := s.MaxPrimeFactors(p.Sub(p, b.NewInt(1)))
	if q.Cmp(found) != 0 {
		t.Error(fmt.Sprintf("Wrong prime factor found: %v q: %v", found, q))
	}

	p, q = s.FrederiksMaxPrimeFactor(1024, 8)
	fmt.Printf("Test: %v\n", new(b.Int).Mod(new(b.Int).Sub(p, b.NewInt(1)), q))
	fmt.Printf("return p: %v q: %v\n", p, q)
}

func TestCommitmentValues(t *testing.T) {
	a := s.NewActiveSecretSharing(6)
	a.PrimeP = b.NewInt(23)
	a.PrimeQ = b.NewInt(11)
	//a.PrimeP, a.PrimeQ = s.FrederiksMaxPrimeFactor(512, 8)
	g, h := a.SetCommitmentValues()
	if g.Cmp(b.NewInt(2)) != 0 && h.Cmp(b.NewInt(3)) != 0 {
		t.Error(fmt.Sprintf("Wrong commitment values found: g: %v, h: %v", g, h))
	}
}

func TestBC(t *testing.T) {
	a := s.NewActiveSecretSharing(6)
	a.PrimeP = b.NewInt(23)
	a.PrimeQ = b.NewInt(11)
	fmt.Println(a.SetCommitmentValues())
	proof := []int64{
		a.BC(b.NewInt(7), b.NewInt(10)).Int64(),
		a.BC(b.NewInt(4), b.NewInt(1)).Int64(),
		a.BC(b.NewInt(1), b.NewInt(4)).Int64(),
	}
	if check := []int64{12, 2, 1}; !reflect.DeepEqual(proof, check) {
		t.Error(fmt.Sprintf("Wrong proof calculated with BC: proof: %v != %v", proof, check))
	}
}

// Test integrations between Active secret sharing module methods.
func TestSSIntegration(t *testing.T) {
	a1 := s.NewActiveSecretSharing(6)
	a1.PrimeP = b.NewInt(23)
	a1.PrimeQ = b.NewInt(11)
	a1.SetCommitmentValues()

	a2 := s.NewActiveSecretSharing(6)
	a2.PrimeP = b.NewInt(23)
	a2.PrimeQ = b.NewInt(11)
	a2.SetCommitmentValues()

	share1 := a1.SecretGen(b.NewInt(3))
	share2 := a1.SecretGen(b.NewInt(7))

	for _, s := range share1 {
		if !a1.VerifyShare(s) || !a2.VerifyShare(s) {
			t.Error("Verification failed for share1, try running with DEBUG=2")
		}
	}
	for _, s := range share2 {
		if !a1.VerifyShare(s) || !a2.VerifyShare(s) {
			t.Error("Verification failed for share2, try running with DEBUG=2")
		}
	}


}

func TestVerifySecret(t *testing.T) {
	a := s.NewActiveSecretSharing(6)
	a.PrimeP = b.NewInt(23)
	a.PrimeQ = b.NewInt(11)
	a.SetCommitmentValues()

	share1 := f.Share{
		S:     *b.NewInt(1),
		T:     *b.NewInt(4),
		Point: 1,
		Id:    uuid.UUID{},
		Proof: []b.Int{*b.NewInt(12), *b.NewInt(2), *b.NewInt(1)},
	}

	share2 := f.Share{
		S:     *b.NewInt(3),
		T:     *b.NewInt(9),
		Point: 3,
		Id:    uuid.UUID{},
		Proof: []b.Int{*b.NewInt(8), *b.NewInt(18), *b.NewInt(18)},
	}

	if !a.VerifyShare(share1) {
		t.Error("Did not verify share1, try running with DEBUG=2")
	}
	if !a.VerifyShare(share2) {
		t.Error("Did not verify share2, try running with DEBUG=2")
	}
}




