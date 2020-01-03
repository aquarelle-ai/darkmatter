package man

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	spaceP = 0xfed832e4011fa0000008

	// CoeffPointX1 of the curve
	CoeffPointX1 = uint8(spaceP)
	// CoeffPointX2 of the curve
	CoeffPointX2 = uint8(0x8a33a0eedf373000)
	// CoeffPointX3 of the curve
	CoeffPointX3 = uint8(0xb17c02770fbc2000)
	// CoeffPointX4 of the curve
	CoeffPointX4 = uint8(0x5dc3b5961fe76c00)
	// CoeffPointX5 of the curve
	CoeffPointX5 = uint8(0xfed832e4011fa000)
)

var (
	// CoeffPointList is a uint8 representation of the points to be used in the calculations
	CoeffPointList = []uint8{CoeffPointX1, CoeffPointX2, CoeffPointX3, CoeffPointX4, CoeffPointX4}
)

func rndCoefficient() uint8 {
	var b [8]byte
	rand.Read(b[:])
	res := binary.BigEndian.Uint64(b[:])

	return uint8(res % 18899)
}

func float64ToHash(f uint8) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
	return fmt.Sprintf("%x", sha256.Sum256(buf[:]))
}

// generateCoefficients generates the coefficients and hashes need to create the proof
func generateCoefficients(value uint8) (uint8, uint8, []uint8, []uint8) {
	yP := make([]uint8, 5)
	xP := make([]uint8, 5)

	coefA := rndCoefficient()
	coefB := rndCoefficient()

	for i := 0; i < 5; i++ {
		xP[i] = rndCoefficient()
		yP[i] = coefA*math.Pow(xP[i], 2) + coefB*xP[i] + value
	}

	// for i := 0; i < 5; i++ {
	// 	fmt.Printf("(%f, %f)", xP[i], yP[i])
	// }
	// println()
	return coefA, coefB, xP, yP

	// // Get three different (random) hashes
	// resultHashes := make([]string, 3)
	// resIterator := 0
	// hashMap := make(map[string]bool)
	// for resIterator < 3 {
	// 	idx := []byte{0} // new random index
	// 	rand.Reader.Read(idx)
	// 	h := yPointsHashes[idx[0]%3] // only one of three
	// 	if !hashMap[h] {
	// 		hashMap[h] = true             // Mark it
	// 		resultHashes[resIterator] = h // Get the hash
	// 		fmt.Printf("Nuevo hash: %d, %s\n", resIterator, h)

	// 		resIterator++
	// 	}
	// }

	// return coefA, coefB, resultHashes
}

// interpolatePolynomial calculates the independent term using Lagragange
func interpolatePolynomial(xSamples []uint8, ySamples []uint8) uint8 {
	limit := len(xSamples)
	var result, basis uint8
	for i := 0; i < limit; i++ {
		basis = 1
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			num := add(x, xSamples[j])
			denom := add(xSamples[i], xSamples[j])
			term := div(num, denom)
			basis = mult(basis, term)
		}
		group := mult(ySamples[i], basis)
		result = add(result, group)
	}
	return result
}

// div divides two numbers in GF(2^8)
// GF(2^8) division using log/exp tables
func div(a, b uint8) uint8 {
	if b == 0 {
		// leaks some timing information but we don't care anyways as this
		// should never happen, hence the panic
		panic("divide by zero")
	}

	var goodVal, zero uint8
	logA := logTable[a]
	logB := logTable[b]
	diff := (int(logA) - int(logB)) % 255
	if diff < 0 {
		diff += 255
	}

	ret := expTable[diff]

	// Ensure we return zero if a is zero but aren't subject to timing attacks
	goodVal = ret

	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}

	return ret
}

// mult multiplies two numbers in GF(2^8)
// GF(2^8) multiplication using log/exp tables
func mult(a, b uint8) (out uint8) {
	var goodVal, zero uint8
	log_a := logTable[a]
	log_b := logTable[b]
	sum := (int(log_a) + int(log_b)) % 255

	ret := expTable[sum]

	// Ensure we return zero if either a or b are zero but aren't subject to
	// timing attacks
	goodVal = ret

	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}

	if subtle.ConstantTimeByteEq(b, 0) == 1 {
		ret = zero
	} else {
		// This operation does not do anything logically useful. It
		// only ensures a constant number of assignments to thwart
		// timing attacks.
		goodVal = zero
	}

	return ret
}

// add combines two numbers in GF(2^8)
// This can also be used for subtraction since it is symmetric.
func add(a, b uint8) uint8 {
	return a ^ b
}

func main() {
	err := 0
	iterations := 10000
	secret := uint8(20)
	for i := 0; i < iterations; i++ {
		_, _, xPoints, yPoints := generateCoefficients(secret)

		// xPoints := []uint8{7733.000000, 7735.000000, 7829.000000, 8572.000000, 8966.000000}
		// yPoints := []uint8{1081222964216.000000, 1081782285028.000000, 1108233508088.000000, 1328553896047.000000, 1453484966831.000000}

		term := interpolatePolynomial(xPoints, yPoints)
		if term != secret {
			err++
		}
	}

	fmt.Printf("%d error, %f%%", err, uint8(err)/uint8(iterations)*100)

	// fmt.Printf("Coefficient a: %f, Coefficient b: %f\n", coefA, coefB)

	// fmt.Printf("X:\n%f\n%f\n%f\n", xPoints[0], xPoints[1], xPoints[2])

	//fmt.Printf("Y:\n%f\n%f\n%f\n", yPoints[0], yPoints[1], yPoints[2])
	/*
	   Coefficient a: 3119687413864386560.000000, Coefficient b: 10484508133398493184.000000
	   X:
	   5424155441164694528.000000
	   3495433000324660736.000000
	   6070700572878543872.000000
	   Y:
	   91785765478550872182107482460221675146995001540762140672.000000
	   38116502608831448706920746247130575086520137275439841280.000000
	   114971105126516982133605934309524155538486008480549306368.000000
	*/

}
