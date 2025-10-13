package shazam

import (
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"sort"
)

const (
	dspRatio    = 4
	freqBinSize = 1024
	maxFreq     = 10000.0 // 10kHz
	minFreq     = 80.0    // remove low frequency rumble from live input
	hopSize     = freqBinSize / 32

	preEmphasisCoeff     = 0.97
	targetRMSLevel       = 0.18
	maxNormalizationGain = 12.0
	noiseGateMultiplier  = 2.5
	minMagnitudeDB       = -85.0
	peakProminenceDB     = 7.5
	maxPeaksPerFrame     = 6
	epsilon              = 1e-10
)

func Spectrogram(sample []float64, sampleRate int) ([][]complex128, error) {
	processedSample := preprocessSample(sample, sampleRate)
	filteredSample := LowPassFilter(maxFreq, float64(sampleRate), processedSample)

	downsampledSample, err := Downsample(filteredSample, sampleRate, sampleRate/dspRatio)
	if err != nil {
		return nil, fmt.Errorf("couldn't downsample audio sample: %v", err)
	}

	if len(downsampledSample) < freqBinSize {
		padded := make([]float64, freqBinSize)
		copy(padded, downsampledSample)
		downsampledSample = padded
	}

	step := freqBinSize - hopSize
	if step <= 0 {
		step = freqBinSize
	}

	numOfWindows := 1 + (len(downsampledSample)-freqBinSize)/hopSize
	if numOfWindows <= 1 {
		numOfWindows = len(downsampledSample) / step
	}
	if numOfWindows < 1 {
		numOfWindows = 1
	}

	spectrogram := make([][]complex128, numOfWindows)

	window := make([]float64, freqBinSize)
	for i := range window {
		window[i] = 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/(float64(freqBinSize)-1))
	}

	// Perform STFT
	for i := 0; i < numOfWindows; i++ {
		start := i * hopSize
		end := start + freqBinSize
		if end > len(downsampledSample) {
			end = len(downsampledSample)
		}

		bin := make([]float64, freqBinSize)
		copy(bin, downsampledSample[start:end])

		// Apply Hamming window
		for j := range window {
			bin[j] *= window[j]
		}

		spectrogram[i] = FFT(bin)
	}

	return spectrogram, nil
}

func preprocessSample(input []float64, sampleRate int) []float64 {
	if len(input) == 0 {
		return input
	}

	preprocessed := removeDCOffset(input)
	preprocessed = HighPassFilter(minFreq, float64(sampleRate), preprocessed)
	preprocessed = applyPreEmphasis(preprocessed, preEmphasisCoeff)
	preprocessed = applyNoiseGate(preprocessed, noiseGateMultiplier)
	preprocessed = normalizeRMS(preprocessed, targetRMSLevel, maxNormalizationGain)

	return preprocessed
}

// LowPassFilter is a first-order low-pass filter that attenuates high
// frequencies above the cutoffFrequency.
// It uses the transfer function H(s) = 1 / (1 + sRC), where RC is the time constant.
func LowPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
	rc := 1.0 / (2 * math.Pi * cutoffFrequency)
	dt := 1.0 / sampleRate
	alpha := dt / (rc + dt)

	filteredSignal := make([]float64, len(input))
	var prevOutput float64 = 0

	for i, x := range input {
		if i == 0 {
			filteredSignal[i] = x * alpha
		} else {

			filteredSignal[i] = alpha*x + (1-alpha)*prevOutput
		}
		prevOutput = filteredSignal[i]
	}
	return filteredSignal
}

// HighPassFilter removes low-frequency rumble often present in live recordings.
func HighPassFilter(cutoffFrequency, sampleRate float64, input []float64) []float64 {
	rc := 1.0 / (2 * math.Pi * cutoffFrequency)
	dt := 1.0 / sampleRate
	alpha := rc / (rc + dt)

	output := make([]float64, len(input))
	var prevInput float64
	var prevOutput float64

	for i, x := range input {
		output[i] = alpha * (prevOutput + x - prevInput)
		prevInput = x
		prevOutput = output[i]
	}
	return output
}

func removeDCOffset(input []float64) []float64 {
	if len(input) == 0 {
		return input
	}

	sum := 0.0
	for _, v := range input {
		sum += v
	}
	mean := sum / float64(len(input))

	output := make([]float64, len(input))
	for i, v := range input {
		output[i] = v - mean
	}
	return output
}

func applyPreEmphasis(input []float64, coeff float64) []float64 {
	if len(input) == 0 {
		return input
	}
	output := make([]float64, len(input))
	output[0] = input[0]
	for i := 1; i < len(input); i++ {
		output[i] = input[i] - coeff*input[i-1]
	}
	return output
}

func applyNoiseGate(input []float64, multiplier float64) []float64 {
	if len(input) == 0 {
		return input
	}

	absValues := make([]float64, len(input))
	for i, v := range input {
		absValues[i] = math.Abs(v)
	}

	noiseFloor := medianFloat64(absValues)
	if noiseFloor <= 0 {
		return input
	}
	threshold := noiseFloor * multiplier

	output := make([]float64, len(input))
	for i, v := range input {
		if math.Abs(v) < threshold {
			output[i] = 0
			continue
		}
		output[i] = v
	}
	return output
}

func normalizeRMS(input []float64, targetLevel, maxGain float64) []float64 {
	if len(input) == 0 {
		return input
	}

	sumSquares := 0.0
	for _, v := range input {
		sumSquares += v * v
	}
	rms := math.Sqrt(sumSquares / float64(len(input)))
	if rms <= 0 {
		return input
	}

	gain := targetLevel / rms
	if gain > maxGain {
		gain = maxGain
	}
	output := make([]float64, len(input))
	for i, v := range input {
		output[i] = v * gain
	}
	return output
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// Downsample downsamples the input audio from originalSampleRate to targetSampleRate
func Downsample(input []float64, originalSampleRate, targetSampleRate int) ([]float64, error) {
	if targetSampleRate <= 0 || originalSampleRate <= 0 {
		return nil, errors.New("sample rates must be positive")
	}
	if targetSampleRate > originalSampleRate {
		return nil, errors.New("target sample rate must be less than or equal to original sample rate")
	}

	ratio := originalSampleRate / targetSampleRate
	if ratio <= 0 {
		return nil, errors.New("invalid ratio calculated from sample rates")
	}

	var resampled []float64
	for i := 0; i < len(input); i += ratio {
		end := i + ratio
		if end > len(input) {
			end = len(input)
		}

		sum := 0.0
		for j := i; j < end; j++ {
			sum += input[j]
		}
		avg := sum / float64(end-i)
		resampled = append(resampled, avg)
	}

	return resampled, nil
}

type Peak struct {
	Time float64
	Freq complex128
}

// ExtractPeaks analyzes a spectrogram and extracts significant peaks in the frequency domain over time.
func ExtractPeaks(spectrogram [][]complex128, audioDuration float64) []Peak {
	if len(spectrogram) < 1 {
		return []Peak{}
	}

	// Define frequency bands (in terms of FFT bin indices)
	// Wider/log-spaced bands help capture instrument and vocal structure up to 8-10 kHz
	bands := []struct{ min, max int }{
		{0, 20}, {20, 40}, {40, 80}, {80, 160}, {160, 320}, {320, 512}, {512, 1024},
	}

	var peaks []Peak
	binDuration := audioDuration / float64(len(spectrogram))

	for binIdx, bin := range spectrogram {
		magnitudes := make([]float64, len(bin))
		for i, freq := range bin {
			mag := 20 * math.Log10(cmplx.Abs(freq)+epsilon)
			if mag < minMagnitudeDB {
				mag = minMagnitudeDB
			}
			magnitudes[i] = mag
		}

		var frameCandidates []struct {
			peak      Peak
			magnitude float64
		}

		for _, band := range bands {
			start := band.min
			if start >= len(magnitudes) {
				continue
			}
			end := band.max
			if end > len(magnitudes) {
				end = len(magnitudes)
			}
			if end <= start {
				continue
			}

			bandMagnitudes := magnitudes[start:end]
			bandMedian := medianFloat64(bandMagnitudes)
			threshold := math.Max(bandMedian+peakProminenceDB, minMagnitudeDB)

			bestIdx := -1
			bestMag := threshold
			for localIdx, mag := range bandMagnitudes {
				globalIdx := start + localIdx
				if mag < threshold {
					continue
				}
				if !isLocalMaximum(magnitudes, globalIdx) {
					continue
				}
				if mag > bestMag {
					bestMag = mag
					bestIdx = globalIdx
				}
			}

			if bestIdx == -1 {
				continue
			}

			peakTime := float64(binIdx)*binDuration + binDuration/2
			peak := Peak{
				Time: peakTime,
				Freq: complex(float64(bestIdx), bestMag),
			}
			frameCandidates = append(frameCandidates, struct {
				peak      Peak
				magnitude float64
			}{peak: peak, magnitude: bestMag})
		}

		if len(frameCandidates) > maxPeaksPerFrame {
			sort.Slice(frameCandidates, func(i, j int) bool {
				return frameCandidates[i].magnitude > frameCandidates[j].magnitude
			})
			frameCandidates = frameCandidates[:maxPeaksPerFrame]
		}

		for _, candidate := range frameCandidates {
			peaks = append(peaks, candidate.peak)
		}
	}

	return peaks
}

func isLocalMaximum(magnitudes []float64, idx int) bool {
	radius := 2
	target := magnitudes[idx]
	for offset := -radius; offset <= radius; offset++ {
		if offset == 0 {
			continue
		}
		neighborIdx := idx + offset
		if neighborIdx < 0 || neighborIdx >= len(magnitudes) {
			continue
		}
		if magnitudes[neighborIdx] > target {
			return false
		}
	}
	return true
}
