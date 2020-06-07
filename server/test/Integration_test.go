package test

import (
	"../framework"
	"../src"
	"math/big"
	"testing"
)

func TestSecretSharingAdd(t *testing.T) {
	var ss = src.NewSecretSharing(5, 11)
	sa := src.SimpleArithmetic{}

	input1 := ss.SecretGen(big.NewInt(4))
	input2 := ss.SecretGen(big.NewInt(5))

	src.Debug(1, "Share 1: %v\n", input1)
	src.Debug(1, "Share 2: %v\n", input2)

	// Add
	for i, _ := range input1 {
		input1[i] = sa.Add(input1[i], input2[i])
	}

	src.Debug(1, "Added secrets: %v\n", input1)

	recon := ss.Reconstruct(input1)

	if recon.Cmp(big.NewInt(9)) != 0 {
		t.Errorf("Found incorrect reconstructed result: %s, expected 9", recon.String())
	}

}

func TestSecretSharingMinus(t *testing.T) {
	var ss = src.NewSecretSharing(5, 11)
	sa := src.SimpleArithmetic{}

	input1 := ss.SecretGen(big.NewInt(4))
	input2 := ss.SecretGen(big.NewInt(-3))

	src.Debug(1, "Share 1: %v\n", input1)
	src.Debug(1, "Share 2: %v\n", input2)

	// Add
	for i, _ := range input1 {
		input1[i] = sa.Add(input1[i], input2[i])
	}

	src.Debug(1, "Added secrets: %v\n", input1)

	recon := ss.Reconstruct(input1)

	if recon.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Found incorrect reconstructed result: %s, expected 9", recon.String())
	}

}

func TestSecretSharingAddSequence(t *testing.T) {
	var ss = src.NewSecretSharing(5, 11)
	sa := src.SimpleArithmetic{}

	input1 := ss.SecretGen(big.NewInt(1))
	input2 := ss.SecretGen(big.NewInt(2))
	input3 := ss.SecretGen(big.NewInt(3))
	input4 := ss.SecretGen(big.NewInt(4))
	inputs := [][]framework.Share{input1, input2, input3, input4}

	src.Debug(1, "Share 1: %v\n", input1)
	src.Debug(1, "Share 2: %v\n", input2)
	src.Debug(1, "Share 3: %v\n", input3)
	src.Debug(1, "Share 4: %v\n", input4)

	// Add
	result := ss.SecretGen(big.NewInt(0))
	for i := 0; i <= 3; i++ {
		for j, _ := range input1 {
			if i == 0 {
				result[j] = inputs[i][j]
			} else {
				result[j] = sa.Add(result[j], inputs[i][j])
			}
		}
		src.Debug(1, "Result after addition %d: %v\n", i, result)
	}

	src.Debug(1, "Added secrets: %v\n", result)

	recon := ss.Reconstruct(result)

	if recon.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Found incorrect reconstructed result: %s, expected 1+2+3+4 = 10", recon.String())
	}

}

// TODO: Check om vi skal lave test som viser at multiplication ikke kan laves i sequence pga. 2t polynomial

func TestSecretSharingMult(t *testing.T) {
	var ss = src.NewSecretSharing(5, 179)
	sa := src.SimpleArithmetic{}

	input1 := ss.SecretGen(big.NewInt(2))
	input2 := ss.SecretGen(big.NewInt(5))

	src.Debug(1, "Share 1: %v\n", input1)
	src.Debug(1, "Share 2: %v\n", input2)

	// Multiply
	for i, _ := range input1 {
		input1[i] = sa.Multiply_f(input1[i], input2[i])
	}

	src.Debug(1, "Added secrets: %v\n", input1)

	recon := ss.Reconstruct(input1, 2)

	if recon.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("Found incorrect reconstructed result: %s, expected 10", recon.String())
	}

}
