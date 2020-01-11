package futurepia_txsigner

import (
	"errors"
	"github.com/blocktree/go-owcrypt"
)



func equals(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for index := 0; index < len(a); index++ {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func makeCompact(sig, publicKey, hash []byte) ([]byte, error) {
	for i := 0; i < 2; i++ {
		tmp := append(sig, byte(i))
		pk, ret := owcrypt.RecoverPubkey(tmp, hash, owcrypt.ECC_CURVE_SECP256K1)
		if ret == owcrypt.SUCCESS && equals(pk, publicKey) {
			result := make([]byte, 1, 2*32+1)
			result[0] = 27 + byte(i)
			result[0] += 4
			// Not sure this needs rounding but safer to do so.
			curvelen := (256 + 7) / 8

			// Pad R and S to curvelen if needed.
			bytelen := (256 + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig[:32]...)

			bytelen = (256 + 7) / 8
			if bytelen < curvelen {
				result = append(result,
					make([]byte, curvelen-bytelen)...)
			}
			result = append(result, sig[32:]...)

			return result, nil
		}
	}

	return nil, errors.New("no valid solution for pubkey found")
}

func isCanonical(compactSig []byte) bool {
	d := compactSig
	t1 := (d[1] & 0x80) == 0
	t2 := !(d[1] == 0 && ((d[2] & 0x80) == 0))
	t3 := (d[33] & 0x80) == 0
	t4 := !(d[33] == 0 && ((d[34] & 0x80) == 0))
	return t1 && t2 && t3 && t4
}
