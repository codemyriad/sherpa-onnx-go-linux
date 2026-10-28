# Cassini release maintenance

This branch, `v2.0.1-cassini`, is upstream `sherpa-onnx-go-linux` **v2.0.1**
(`69bcfdb`) with Cassini's native libraries. Tags are `v2.0.1-cassini.N` and are
never moved. The module path stays `github.com/k2-fsa/sherpa-onnx-go-linux/v2`,
so consumers use it through a `replace`:

```
require github.com/k2-fsa/sherpa-onnx-go/v2 v2.0.1
replace github.com/k2-fsa/sherpa-onnx-go-linux/v2 => github.com/codemyriad/sherpa-onnx-go-linux/v2 v2.0.1-cassini.1
```

The `v1.13.7-cassini` branch and its tags (`v1.13.7-cassini.1` … `.8`) are
unchanged and keep serving the v1 module path.

## v2.0.1-cassini.1

**Native source:** codemyriad/sherpa-onnx tag `v2.0.1-cassini-1` (annotated tag
object `82f5f0122ecd69c46ec5bb5b72e4f317c4f071fa`, commit
[`c810dfcf409b2a72a13e9f0ac961ff55473e350d`](https://github.com/codemyriad/sherpa-onnx/commit/c810dfcf409b2a72a13e9f0ac961ff55473e350d)):
upstream sherpa-onnx v2.0.1 plus k2-fsa/sherpa-onnx#4032 (Nemotron-3-Diarization
/ Sortformer, head `6a4ad68b`), TDT detection from the NeMo joiner output size
(plain-URL Parakeet Ultra exports load), and the Cassini version marker. See
that tree's `CASSINI.md`. Runtime version (`GetVersion()`):

```
2.0.1+cassini.parakeet-tdt-joiner-v1.nemotron-diarization-v2
```

The 1.13.7 markers `+cassini-parakeet-v3-reference-v1` and `.parakeet-ultra-v1`
are gone with the reference frontend (upstream #3999, in v2.0.1, fixes quiet
audio instead). `.nemotron-diarization-v2` keeps its text.

**Go source.** `sherpa_onnx.go` and `c-api.h` are copied from that tree
(`scripts/go/sherpa_onnx.go`, `sherpa-onnx/c-api/c-api.h`). The only change from
upstream v2.0.1 is #4032's member
`OfflineSpeakerSegmentationModelConfig.Sortformer` (`Model`, `Threshold`; 0
means 0.5), C `SherpaOnnxOfflineSpeakerSegmentationSortformerModelConfig
sortformer` at the end of `SherpaOnnxOfflineSpeakerSegmentationModelConfig`.
Pass the Nemotron model as `Segmentation.Sortformer.Model`; the 1.13.7 forks'
`Segmentation.Pyannote.Model` shim is gone. The stock
`github.com/k2-fsa/sherpa-onnx-go/v2` aliases reach the new field unchanged.

**Libraries.**

| Directory | Files | Origin |
|---|---|---|
| `lib/x86_64-unknown-linux-gnu` | C API, C++ API | built by `cassini-native.yml` on `ubuntu-24.04` |
| `lib/aarch64-unknown-linux-gnu` | C API, C++ API | built by `cassini-native.yml` on `ubuntu-24.04-arm` |
| both of the above | `libonnxruntime.so` | ONNX Runtime **1.28.2**, the archive upstream v2.0.1 pins (`cmake/onnxruntime-linux-{x86_64,aarch64}.cmake`); byte-identical to the stock v2.0.1 module's, checked by `cmp` in the build job |
| `lib/arm-unknown-linux-gnueabihf` | all three | **stock upstream v2.0.1, unchanged** |

The C++ API is rebuilt too, because it links `libsherpa-onnx-c-api.so` and
#4032 changes the diarization config layout it passes in.

**Linux arm32 is stock v2.0.1.** It has #3999 (quiet audio decodes), but not
the joiner TDT detection (plain-URL Parakeet Ultra still exits at load) and not
Sortformer. Its runtime version is plain `2.0.1`. With this module's Go wrapper
the C diarization config has #4032's layout, which the stock arm32 library does
not, so **never call `NewOfflineSpeakerDiarization` on arm32**: gate diarization
and Ultra on the `.nemotron-diarization-v2` / `.parakeet-tdt-joiner-v1`
markers, as Cassini does. Its ONNX Runtime is the pinned
`onnxruntime-linux-arm-1.28.2.zip` (sha256 `a8b4a3f1…`), byte-identical.

**Build.** `.github/workflows/cassini-native.yml` (amd64 and arm64 matrix,
`golang:1.24-bookworm` container: Debian Bookworm, glibc 2.36, GCC 12) runs
`scripts/build-native.sh cpu`, which downloads the codeload tarball of
`c810dfcf` (sha256 `28122478…`, checked), builds the `sherpa-onnx-c-api` and
`sherpa-onnx-cxx-api` targets with diarization on and TTS off, and copies the
ORT library from the pinned archive. The job then checks NEEDED (only the C/C++
runtime, ONNX Runtime, and the C API for the C++ API), RUNPATH `$ORIGIN`, the
highest glibc symbol (at most `GLIBC_2.34`), the runtime version string, and
`cmp`s ONNX Runtime against the committed one. Rebuild locally with
`scripts/Dockerfile.native` (as in the v1.13.7 notes) if needed.

**Probe** (`scripts/cassini-probe.sh`, on each runner, through this module's
Go wrapper with the build's libraries copied into `lib/`; inputs pinned by
sha256):

- version string equal to the one above;
- public `en.wav` (3.85 s) at gain 1 and 0.001 (-60 dB), greedy, with
  Cassini's production `parakeet-tdt-0.6b-v3-int8` and
  `parakeet-ultra-0.6b-int8` bundles from dist.gocassini.com (Ultra encoder
  `url = https://huggingface.co/moondream/parakeet-ultra`, which stock v2.0.1
  rejects with `vocab_size: 8193 != output_size: 8198` and exit 255): each
  must give the full sentence;
- Sortformer diarization of k2-fsa's public `0-four-speakers-zh.wav`
  (56.9 s) with the production `nemotron-3-diarization-int8` model
  (interface version 2, sha256 `47c221ea…`) through
  `Segmentation.Sortformer.Model`: must find 4 speakers.

`.github/workflows/cassini-verify.yml` reruns `sha256sum -c
CASSINI-SHA256SUMS` and the probe on the committed libraries for every
`v2.0.1-cassini.*` tag, on both runners.

**Release run.** The committed amd64 and arm64 libraries are the
`cassini-native-amd64` / `cassini-native-arm64` artifacts of
[run 40706763512](https://github.com/codemyriad/sherpa-onnx-go-linux/actions/runs/40706763512)
(commit `12a45d0`, both jobs green). The earlier
[run 40704823714](https://github.com/codemyriad/sherpa-onnx-go-linux/actions/runs/40704823714)
built the same source with the same `build-native.sh`; its arm64 job passed
and produced byte-identical arm64 libraries, and its amd64 job stopped
before building (`gcc --version | head -1` hit SIGPIPE under `pipefail`,
fixed in `12a45d0`). On both runners:

| Check | amd64 | arm64 |
|---|---|---|
| Runtime version | `2.0.1+cassini.parakeet-tdt-joiner-v1.nemotron-diarization-v2` | same |
| Highest glibc symbol, C API / C++ API | `GLIBC_2.34` / `GLIBC_2.32` | same |
| ONNX Runtime 1.28.2 = stock v2.0.1 module's | yes (`4b3607ae…`) | yes (`cfc800d4…`) |
| Parakeet v3 int8, gain 1 / 0.001 | 27 / 27 tokens, full sentence | 27 / 27, full sentence |
| Parakeet Ultra int8, plain URL, gain 1 / 0.001 | 27 / 27 tokens, full sentence | 27 / 27, full sentence |
| Sortformer, four-speaker sample | 9 turns, 4 speakers | 9 turns, 4 speakers (boundaries within 10 ms of amd64) |

The amd64 libraries were also probed on an x86_64 Linux host after copying
them into `lib/`, with the same result. The same probe against stock v2.0.1
libraries fails on the version and exits 255 at the Ultra bundle.

`CASSINI-SHA256SUMS` records every library in `lib/`. Verify with
`sha256sum -c CASSINI-SHA256SUMS` from the repository root.

Not covered by this module: CUDA (Cassini builds its CUDA libraries from the
same source tag in its own image), fp32 Ultra and fp32 diarization models,
beam search, and runtime checks on arm32.

Never move published Go tags: publish a new `v2.0.1-cassini.N` and update the
consumer's `go.sum`.

## v1.13.7-cassini.*

See `CASSINI.md` on the `v1.13.7-cassini` branch.
