package shazam

import (
	"math"
	"song-recognition/models"
)

const (
	maxFreqBits            = 9
	maxDeltaBits           = 14
	targetZoneSize         = 12 // this densifies fingerprints per anchor, improving match robustness.
	freqQuantizationFactor = freqBinSize / (1 << maxFreqBits)
)

// Fingerprint generates fingerprints from a list of peaks and stores them in an array.
// Each fingerprint consists of an address and a couple.
// The address is a hash. The couple contains the anchor time and the song ID.
func Fingerprint(peaks []Peak, songID uint32) map[uint32]models.Couple {
	fingerprints := map[uint32]models.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			address, ok := createAddress(anchor, target)
			if !ok {
				continue
			}
			anchorTimeMs := uint32(math.Round(anchor.Time * 1000))

			fingerprints[address] = models.Couple{anchorTimeMs, songID}
		}
	}

	return fingerprints
}

// createAddress generates a unique address for a pair of anchor and target points.
// The address is a 32-bit integer where certain bits represent the frequency of
// the anchor and target points, and other bits represent the time difference (delta time)
// between them. This function combines these components into a single address (a hash).
func createAddress(anchor, target Peak) (uint32, bool) {
	if target.Time <= anchor.Time {
		return 0, false
	}

	anchorFreq := quantizeFrequency(real(anchor.Freq))
	targetFreq := quantizeFrequency(real(target.Freq))
	if anchorFreq < 0 || targetFreq < 0 {
		return 0, false
	}

	deltaMs := int(math.Round((target.Time - anchor.Time) * 1000))
	if deltaMs <= 0 {
		return 0, false
	}

	anchorMask := (1 << maxFreqBits) - 1
	targetMask := (1 << maxFreqBits) - 1
	deltaMask := (1 << maxDeltaBits) - 1

	if anchorFreq > anchorMask {
		anchorFreq = anchorMask
	}
	if targetFreq > targetMask {
		targetFreq = targetMask
	}
	if deltaMs > deltaMask {
		deltaMs = deltaMask
	}

	address := uint32(anchorFreq<<(maxDeltaBits+maxFreqBits)) |
		uint32(targetFreq<<maxDeltaBits) |
		uint32(deltaMs&deltaMask)

	return address, true
}

func quantizeFrequency(freq float64) int {
	if math.IsNaN(freq) || math.IsInf(freq, 0) {
		return -1
	}

	value := int(math.Round(freq))
	if value < 0 {
		value = 0
	}

	if freqQuantizationFactor > 1 {
		value = value / freqQuantizationFactor
	}

	return value
}
