package test

import (
	"../framework"
	"../src"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleAdd(t *testing.T) {
	secA := framework.NewSecret(1, 3)
	secB := framework.NewSecret(1, 2)
	sc := src.SimpleArithmetic{}
	sec := sc.Add(secA, secB)

	if sec.PointValue != 5 {
		t.Error("Add(sec(3), sec(2)), got %D, expected 5", sec.PointValue)
	}
}

func TestValidatePoints(t *testing.T) {
	secA := framework.NewSecret(1, 3)
	secB := framework.NewSecret(2, 2)
	sc := src.SimpleArithmetic{}
	assert.Panics(t, func() { sc.Add(secA, secB) }, "Code did not panic on two differing points in add")

}

func TestSimpleMultiplicationConst(t *testing.T) {
	secA := framework.NewSecret(1, 69)
	i := 420
	sc := src.SimpleArithmetic{}
	sec := sc.Multiply_const(secA, i)
	if sec.PointValue != 28980 {
		t.Error("MultConst(sec(69),420), got %D, expected 28980", sec.PointValue)
	}
}
