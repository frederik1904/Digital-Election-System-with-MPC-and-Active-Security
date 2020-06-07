package test

import (
	"../framework"
	"../src"
	"math/big"
	"testing"
)

func TestInit(t *testing.T) {
	var secretsharing = src.NewSecretSharing(5, 179)
	if secretsharing.T != 2 {
		t.Error("Incorrect corrupted parties calc. Expected '2', found '%D'", secretsharing.T)
	}
	if secretsharing.Parties != 5 {
		t.Error("Got wrong number of parties in Init. Expected: '5', Found: '%D'", secretsharing.Parties)
	}
}

func TestSecretGen(t *testing.T) {
	var secretsharing = src.NewSecretSharing(5, 11)

	secrets5 := secretsharing.SecretGen(big.NewInt(5))
	secrets6 := secretsharing.SecretGen(big.NewInt(6))

	if len(secrets5) != 5 || len(secrets6) != 5 {
		t.Error("Not enough secrets generated for parties.")
	}
}

func TestLagrange(t *testing.T) {

	points := []*big.Int{big.NewInt(3), big.NewInt(4), big.NewInt(5)}
	checks := []*big.Int{big.NewInt(10), big.NewInt(7), big.NewInt(6)}

	for i, v := range points {
		tmp := make([]*big.Int, len(points))
		copy(tmp, points)
		j := v
		m := append(tmp[:i], tmp[i+1:]...)

		res := src.Lagrange(j, big.NewInt(11), m)
		if res.Cmp(checks[i]) != 0 {
			t.Errorf("Not correct interpolation constant found. Found %s, expected %s", res, checks[i])
		}

	}
}

func TestReconstruction(t *testing.T) {
	secrets := []framework.Secret{
		framework.NewSecret(1, 1),
		framework.NewSecret(2, 8),
		framework.NewSecret(3, 6),
		framework.NewSecret(4, 6),
		framework.NewSecret(5, 8),
	}

	secretsharing := src.NewSecretSharing(5, 11)
	recon := secretsharing.Reconstruct(secrets)
	if recon.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("Wrong reconstruction. Found %s, expected 7", recon)
	}

	secrets_subset135 := []framework.Secret{secrets[0], secrets[2], secrets[4]}
	recon2 := secretsharing.Reconstruct(secrets_subset135)
	if recon2.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("Wrong reconstruction. Found %s, expected 7", recon2)
	}
}
