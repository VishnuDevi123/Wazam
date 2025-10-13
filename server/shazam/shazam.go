//go:build !js && !wasm
// +build !js,!wasm

package shazam

import (
	"fmt"
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

	for songID, points := range scores {
		song, songExists, err := database.GetSongByID(songID)
		if err != nil {
			logger.Info(fmt.Sprintf("failed to get song by ID (%v): %v", songID, err))
			continue
		}
		if !songExists {
			logger.Info(fmt.Sprintf("song with ID (%v) doesn't exist", songID))
			continue
		}

		match := Match{songID, song.Title, song.Artist, song.YouTubeID, timestamps[songID], points}
		matchList = append(matchList, match)
	}

	sort.SliceStable(matchList, func(i, j int) bool {
		return matchList[i].Score > matchList[j].Score
	})

	return matchList, time.Since(startTime), nil
}

// analyzeRelativeTiming aligns the sample and DB anchor times
// using a histogram of time offsets. The song with the most
// consistent offset alignment scores highest.
func analyzeRelativeTiming(matches map[uint32][][2]uint32) map[uint32]float64 {
	const offsetBinWidth = 20 // milliseconds
	scores := make(map[uint32]float64)

	for songID, pairs := range matches {
		offsetHistogram := make(map[int]int)

		for _, pair := range pairs {
			offset := int(pair[1]) - int(pair[0])
			offsetBin := (offset / offsetBinWidth) * offsetBinWidth
			offsetHistogram[offsetBin]++
		}

		maxCount := 0
		for _, count := range offsetHistogram {
			if count > maxCount {
				maxCount = count
			}
		}

		scores[songID] = float64(maxCount)
	}

	return scores
}