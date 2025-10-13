//go:build !js && !wasm
// +build !js,!wasm

package shazam

import (
	"fmt"
	"math"
	"song-recognition/db"
	"song-recognition/utils"
	"sort"
	"time"
)

type Match struct {
	SongID     uint32
	SongTitle  string
	SongArtist string
	YouTubeID  string
	Timestamp  uint32
	Score      float64
}

// FindMatches analyzes the audio sample to find matching songs in the database.
func FindMatches(audioSample []float64, audioDuration float64, sampleRate int) ([]Match, time.Duration, error) {
	startTime := time.Now()

	spectrogram, err := Spectrogram(audioSample, sampleRate)
	if err != nil {
		return nil, time.Since(startTime), fmt.Errorf("failed to get spectrogram: %v", err)
	}

	peaks := ExtractPeaks(spectrogram, audioDuration)
	sampleFingerprint := Fingerprint(peaks, utils.GenerateUniqueID())

	sampleFingerprintMap := make(map[uint32]uint32)
	for address, couple := range sampleFingerprint {
		sampleFingerprintMap[address] = couple.AnchorTimeMs
	}

	matches, _, err := FindMatchesFGP(sampleFingerprintMap)
	if err != nil {
		return nil, time.Since(startTime), fmt.Errorf("failed to find matches: %v", err)
	}

	return matches, time.Since(startTime), nil
}

// FindMatchesFGP uses the sample fingerprint to find matching songs in the database.
func FindMatchesFGP(sampleFingerprint map[uint32]uint32) ([]Match, time.Duration, error) {
	startTime := time.Now()
	logger := utils.GetLogger()
	const (
		minConsistentMatches = 2
		maxReturnedMatches   = 5
	)

	addresses := make([]uint32, 0, len(sampleFingerprint))
	for address := range sampleFingerprint {
		addresses = append(addresses, address)
	}

	database, err := db.NewDBClient()
	if err != nil {
		return nil, time.Since(startTime), err
	}
	defer database.Close()

	couplesByAddress, err := database.GetCouples(addresses)
	if err != nil {
		return nil, time.Since(startTime), err
	}

	matches := map[uint32][][2]uint32{} // songID -> [(sampleTime, dbTime)]
	timestamps := map[uint32]uint32{}   // songID -> earliest timestamp

	for address, couples := range couplesByAddress {
		sampleTime, exists := sampleFingerprint[address]
		if !exists {
			continue
		}
		for _, couple := range couples {
			matches[couple.SongID] = append(matches[couple.SongID],
				[2]uint32{sampleTime, couple.AnchorTimeMs},
			)
			if existingTime, ok := timestamps[couple.SongID]; !ok || couple.AnchorTimeMs < existingTime {
				timestamps[couple.SongID] = couple.AnchorTimeMs
			}
		}
	}

	scores := analyzeRelativeTiming(matches)
	var matchList []Match

	for songID, summary := range scores {
		if summary.MatchCount < minConsistentMatches {
			continue
		}

		song, songExists, err := database.GetSongByID(songID)
		if err != nil {
			logger.Info(fmt.Sprintf("failed to get song by ID (%v): %v", songID, err))
			continue
		}
		if !songExists {
			logger.Info(fmt.Sprintf("song with ID (%v) doesn't exist", songID))
			continue
		}

		timestamp := timestamps[songID]
		if summary.ReferenceTimestamp != math.MaxUint32 {
			timestamp = summary.ReferenceTimestamp
		}

		match := Match{
			SongID:     songID,
			SongTitle:  song.Title,
			SongArtist: song.Artist,
			YouTubeID:  song.YouTubeID,
			Timestamp:  timestamp,
			Score:      summary.Score,
		}
		matchList = append(matchList, match)
	}

	sort.SliceStable(matchList, func(i, j int) bool {
		if matchList[i].Score == matchList[j].Score {
			return matchList[i].Timestamp < matchList[j].Timestamp
		}
		return matchList[i].Score > matchList[j].Score
	})

	if len(matchList) > maxReturnedMatches {
		matchList = matchList[:maxReturnedMatches]
	}

	return matchList, time.Since(startTime), nil
}

// analyzeRelativeTiming aligns the sample and DB anchor times
// using a histogram of time offsets. The song with the most
// consistent offset alignment scores highest.
func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]alignmentSummary {
	const offsetBinWidth = 18 // milliseconds
	summaries := make(map[uint32]alignmentSummary)

	for songID, pairs := range matches {
		if len(pairs) == 0 {
			continue
		}

		buckets := make(map[int][][2]uint32)

		for _, pair := range pairs {
			offset := int(pair[1]) - int(pair[0])
			offsetBin := int(math.Round(float64(offset)/float64(offsetBinWidth))) * offsetBinWidth
			buckets[offsetBin] = append(buckets[offsetBin], pair)
		}

		totalPairs := len(pairs)
		var best alignmentSummary

		for _, binPairs := range buckets {
			count := len(binPairs)
			if count == 0 {
				continue
			}

			offsets := make([]float64, count)
			earliestAnchor := uint32(math.MaxUint32)

			for i, pair := range binPairs {
				offsets[i] = float64(int(pair[1]) - int(pair[0]))
				if pair[1] < earliestAnchor {
					earliestAnchor = pair[1]
				}
			}

			meanOffset := meanFloat64(offsets)
			spread := stddevFloat64(offsets, meanOffset)

			coverage := float64(count) / float64(totalPairs)
			consistency := 1.0 / (1.0 + spread/float64(offsetBinWidth))
			score := float64(count) * (0.6 + 0.4*coverage) * consistency

			summary := alignmentSummary{
				Score:              score,
				Offset:             int(math.Round(meanOffset)),
				MatchCount:         count,
				ReferenceTimestamp: earliestAnchor,
			}

			if summary.Score > best.Score || (summary.Score == best.Score && summary.MatchCount > best.MatchCount) {
				best = summary
			} else if summary.Score == best.Score && summary.MatchCount == best.MatchCount && summary.ReferenceTimestamp < best.ReferenceTimestamp {
				best = summary
			}
		}

		if best.MatchCount == 0 {
			continue
		}

		summaries[songID] = best
	}

	return summaries
}

type alignmentSummary struct {
	Score              float64
	Offset             int
	MatchCount         int
	ReferenceTimestamp uint32
}

func meanFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stddevFloat64(values []float64, mean float64) float64 {
	switch len(values) {
	case 0:
		return 0
	case 1:
		return math.Abs(values[0] - mean)
	default:
		var sumSquares float64
		for _, v := range values {
			diff := v - mean
			sumSquares += diff * diff
		}
		return math.Sqrt(sumSquares / float64(len(values)-1))
	}
}
