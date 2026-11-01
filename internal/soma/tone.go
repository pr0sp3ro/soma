package soma

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
)

type wavHeader struct {
	RIFF       [4]byte
	Size       uint32
	WAVE       [4]byte
	Format     [4]byte
	FormatSize uint32
	PCM        uint16
	Channels   uint16
	SampleRate uint32
	ByteRate   uint32
	BlockAlign uint16
	Bits       uint16
	Data       [4]byte
	DataSize   uint32
}

// Smooth fades and a quiet second harmonic soften the generated tone.
func createTone(frequency float64, seconds float64) (string, error) {
	const sampleRate int = 44100

	var count int = int(float64(sampleRate) * seconds)
	var samples []int16 = make([]int16, count)

	for index := range samples {
		var position float64 = float64(index) / float64(sampleRate)
		var attack float64 = math.Min(position/0.03, 1)
		var release float64 = math.Min((seconds-position)/0.08, 1)
		var envelope float64 = attack * release * math.Exp(-3*position/seconds)
		var fundamental float64 = math.Sin(2 * math.Pi * frequency * position)
		var harmonic float64 = 0.2 * math.Sin(2*math.Pi*frequency*2*position)

		samples[index] = int16(9000 * envelope * (fundamental + harmonic))
	}

	file, err := os.CreateTemp("", "soma-tone-*.wav")
	if err != nil {
		return "", err
	}

	var path string = file.Name()
	var header = wavHeader{
		RIFF:       [4]byte{'R', 'I', 'F', 'F'},
		Size:       uint32(36 + count*2),
		WAVE:       [4]byte{'W', 'A', 'V', 'E'},
		Format:     [4]byte{'f', 'm', 't', ' '},
		FormatSize: 16,
		PCM:        1,
		Channels:   1,
		SampleRate: uint32(sampleRate),
		ByteRate:   uint32(sampleRate * 2),
		BlockAlign: 2,
		Bits:       16,
		Data:       [4]byte{'d', 'a', 't', 'a'},
		DataSize:   uint32(count * 2),
	}

	err = binary.Write(file, binary.LittleEndian, header)
	if err == nil {
		err = binary.Write(file, binary.LittleEndian, samples)
	}

	err = errors.Join(err, file.Close())
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}
