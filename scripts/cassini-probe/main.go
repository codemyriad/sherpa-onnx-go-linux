// Command cassini-probe checks a Cassini native build through this module's Go
// wrapper (via the stock github.com/k2-fsa/sherpa-onnx-go/v2 aliases):
//
//   - The runtime version must be the Cassini v2.0.1 version string.
//   - Each Parakeet model directory (-asr, repeatable) must transcribe the
//     public English sample both at normal level and attenuated by 60 dB
//     (x0.001). Stock v1.13.7 decodes the attenuated case to nothing; stock
//     v2.0.1 exits at load on a Parakeet Ultra bundle whose encoder url is the
//     plain https://huggingface.co/moondream/parakeet-ultra (no "tdt"), which
//     the Cassini build detects as TDT from the joiner output size.
//   - With -sortformer, the Nemotron-3-Diarization (interface version 2) model
//     is loaded through the new Segmentation.Sortformer member
//     (k2-fsa/sherpa-onnx#4032) and must find at least -min-speakers speakers
//     in -diar-wav.
//
// Usage:
//
//	cassini-probe -wav en.wav -asr DIR [-asr DIR...] [-sortformer MODEL -diar-wav WAV]
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/v2/sherpa_onnx"
)

const runtimeVersion = "2.0.1+cassini.parakeet-tdt-joiner-v1.nemotron-diarization-v2"

var markers = []string{
	"2.0.1+cassini.",
	".parakeet-tdt-joiner-v1",
	".nemotron-diarization-v2",
}

const expected = "ask what you can do for your country"

type dirs []string

func (d *dirs) String() string     { return strings.Join(*d, ",") }
func (d *dirs) Set(v string) error { *d = append(*d, v); return nil }

func readWav(path string) ([]float32, int) {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	sr := 0
	var pcm []float32
	for i := 12; i+8 <= len(b); {
		id := string(b[i : i+4])
		n := int(binary.LittleEndian.Uint32(b[i+4 : i+8]))
		body := b[i+8 : min(len(b), i+8+n)]
		if id == "fmt " {
			sr = int(binary.LittleEndian.Uint32(body[4:8]))
			if binary.LittleEndian.Uint16(body[0:2]) != 1 || binary.LittleEndian.Uint16(body[2:4]) != 1 || binary.LittleEndian.Uint16(body[14:16]) != 16 {
				panic("need mono s16 PCM")
			}
		}
		if id == "data" {
			for j := 0; j+1 < len(body); j += 2 {
				pcm = append(pcm, float32(int16(binary.LittleEndian.Uint16(body[j:])))/32768)
			}
		}
		i += 8 + n + n%2
	}
	return pcm, sr
}

func probeASR(dir string, pcm []float32, sr int) bool {
	ok := true
	cfg := sherpa.OfflineRecognizerConfig{}
	cfg.FeatConfig.SampleRate = 16000
	cfg.FeatConfig.FeatureDim = 128
	cfg.ModelConfig.Transducer.Encoder = dir + "/encoder.int8.onnx"
	cfg.ModelConfig.Transducer.Decoder = dir + "/decoder.int8.onnx"
	cfg.ModelConfig.Transducer.Joiner = dir + "/joiner.int8.onnx"
	cfg.ModelConfig.Tokens = dir + "/tokens.txt"
	cfg.ModelConfig.ModelType = "nemo_transducer"
	cfg.ModelConfig.NumThreads = 4
	cfg.ModelConfig.Provider = "cpu"
	cfg.DecodingMethod = "greedy_search"
	r := sherpa.NewOfflineRecognizer(&cfg)
	if r == nil {
		fmt.Printf("FAIL: %s: recognizer creation failed\n", dir)
		return false
	}
	defer sherpa.DeleteOfflineRecognizer(r)
	for _, gain := range []float32{1, 0.001} {
		x := make([]float32, len(pcm))
		for i, v := range pcm {
			x[i] = v * gain
		}
		s := sherpa.NewOfflineStream(r)
		s.AcceptWaveform(sr, x)
		r.Decode(s)
		res := s.GetResult()
		text := strings.TrimSpace(res.Text)
		fmt.Printf("asr %s gain=%g tokens=%d text=%q\n", dir, gain, len(res.Tokens), text)
		if !strings.Contains(strings.ToLower(text), expected) {
			fmt.Printf("FAIL: %s gain=%g did not transcribe the sample\n", dir, gain)
			ok = false
		}
		sherpa.DeleteOfflineStream(s)
	}
	return ok
}

func probeSortformer(model, wav string, minSpeakers int) bool {
	pcm, sr := readWav(wav)
	cfg := sherpa.OfflineSpeakerDiarizationConfig{MinDurationOn: 0.3, MinDurationOff: 0.5}
	// #4032's member; the Pyannote slot stays empty.
	cfg.Segmentation.Sortformer.Model = model
	cfg.Segmentation.NumThreads = 4
	cfg.Segmentation.Provider = "cpu"
	sd := sherpa.NewOfflineSpeakerDiarization(&cfg)
	if sd == nil {
		fmt.Printf("FAIL: sortformer %s: diarization creation failed\n", model)
		return false
	}
	defer sherpa.DeleteOfflineSpeakerDiarization(sd)
	if got := sd.SampleRate(); got != sr {
		fmt.Printf("FAIL: sortformer expects %d Hz, %s is %d Hz\n", got, wav, sr)
		return false
	}
	segs := sd.Process(pcm)
	speakers := map[int]bool{}
	for _, s := range segs {
		speakers[s.Speaker] = true
		fmt.Printf("diar %.2f-%.2f speaker_%02d\n", s.Start, s.End, s.Speaker)
	}
	fmt.Printf("sortformer %s wav=%s seconds=%.2f segments=%d speakers=%d\n",
		model, wav, float64(len(pcm))/float64(sr), len(segs), len(speakers))
	if len(speakers) < minSpeakers {
		fmt.Printf("FAIL: sortformer found %d speakers, want at least %d\n", len(speakers), minSpeakers)
		return false
	}
	return true
}

func main() {
	wav := flag.String("wav", "", "16 kHz mono s16 English sample for the ASR probe")
	var asr dirs
	flag.Var(&asr, "asr", "Parakeet int8 model directory (repeatable)")
	sortformer := flag.String("sortformer", "", "Nemotron-3-Diarization ONNX model (interface version 2)")
	diarWav := flag.String("diar-wav", "", "16 kHz mono s16 multi-speaker sample for -sortformer")
	minSpeakers := flag.Int("min-speakers", 2, "speakers -sortformer must find in -diar-wav")
	flag.Parse()
	if (len(asr) > 0) != (*wav != "") || (*sortformer != "") != (*diarWav != "") || (len(asr) == 0 && *sortformer == "") {
		flag.Usage()
		os.Exit(2)
	}

	version := sherpa.GetVersion()
	fmt.Printf("sherpa version=%s gitsha=%s\n", version, sherpa.GetGitSha1())
	ok := true
	if version != runtimeVersion {
		fmt.Printf("FAIL: version is %q, want %q\n", version, runtimeVersion)
		ok = false
	}
	for _, m := range markers {
		if !strings.Contains(version, m) {
			fmt.Printf("FAIL: version lacks marker %s\n", m)
			ok = false
		}
	}
	if len(asr) > 0 {
		pcm, sr := readWav(*wav)
		fmt.Printf("wav=%s sr=%d samples=%d\n", *wav, sr, len(pcm))
		for _, d := range asr {
			ok = probeASR(d, pcm, sr) && ok
		}
	}
	if *sortformer != "" {
		ok = probeSortformer(*sortformer, *diarWav, *minSpeakers) && ok
	}
	if !ok {
		os.Exit(1)
	}
	fmt.Println("PASS")
}
