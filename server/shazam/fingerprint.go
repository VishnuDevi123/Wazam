package shazam

import (
	"song-recognition/models"
)

const (
	maxFreqBits    = 9
	maxDeltaBits   = 14
	targetZoneSize = 12 // this densifies fingerprints per anchor, improving match robustness.
)

// Fingerprint generates fingerprints from a list of peaks and stores them in an array.
// Each fingerprint consists of an address and a couple.
// The address is a hash. The couple contains the anchor time and the song ID.
func Fingerprint(peaks []Peak, songID uint32) map[uint32]models.Couple {
	fingerprints := map[uint32]models.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			address := createAddress(anchor, target)
			anchorTimeMs := uint32(anchor.Time * 1000)

			fingerprints[address] = models.Couple{anchorTimeMs, songID}
		}
	}

	return fingerprints
}

// createAddress generates a unique address for a pair of anchor and target points.
// The address is a 32-bit integer where certain bits represent the frequency of
// the anchor and target points, and other bits represent the time difference (delta time)
// between them. This function combines these components into a single address (a hash).
func createAddress(anchor, target Peak) uint32 {
    anchorFreq := int(real(anchor.Freq))
    targetFreq := int(real(target.Freq))
    deltaMs := uint32((target.Time - anchor.Time) * 1000)
    
    // mask fields to fixed bit widths to avoid overflow/collision
    anchorMask := (1 << maxFreqBits) - 1
    targetMask := (1 << maxFreqBits) - 1
    deltaMask  := (1 << maxDeltaBits) - 1

    af := (anchorFreq & anchorMask)
    tf := (targetFreq & targetMask)
    dt := int(deltaMs) & deltaMask

    // Layout: [af | tf | dt]
    address := uint32(af<<(maxDeltaBits+maxFreqBits)) | uint32(tf<<maxDeltaBits) | uint32(dt)
    return address
}
