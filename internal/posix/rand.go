package posix

var rngidx int
var rngvec [32]int
var rngseed int

func Srandom(seed int) {
	rngseed = seed
	if rngseed == 0 {
		rngseed = 1
	}
	rngvec[0] = rngseed
	for i := 1; i < 31; i++ {
		rngvec[i] = (16807 * rngvec[i-1]) % 2147483647
		if rngvec[i] < 0 {
			rngvec[i] = rngvec[i] + 2147483647
		}
	}
	for i := 31; i < 34; i++ {
		rngvec[i%32] = rngvec[(i+1)%32]
	}
	for i := 34; i < 344; i++ {
		rngvec[i%32] = rngvec[(i+1)%32] + rngvec[(i+29)%32]
	}
}

func Random() uint32 {
	if rngseed == 0 {
		Srandom(1)
	}
	rngvec[(rngidx+24)%32] = rngvec[(rngidx+25)%32] + rngvec[(rngidx+21)%32]
	rngidx = (rngidx + 1) % 32
	return uint32(rngvec[(rngidx+23)%32]) >> 1
}

func ResetRandom() {
	rngidx = 0
	Srandom(1)
}
