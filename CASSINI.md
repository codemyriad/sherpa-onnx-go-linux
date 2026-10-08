# Cassini release maintenance

`v1.13.7-cassini.6` rebuilds the amd64/arm64 C API from
[`7e87f9b23838daceb88d96422777851c41c80f99`](https://github.com/codemyriad/sherpa-onnx/commit/7e87f9b23838daceb88d96422777851c41c80f99),
retained by the `v1.13.7-cassini-nemotron-v2` source tag. This backports the
reviewed Nemotron final-chunk masking fix, strict model metadata validation
and cache hardening. It requires ONNX interface version 2 (`num_frames` input)
and reports `.nemotron-diarization-v2`. Use the current fp32/int8 Nemotron
exports in the [distribution manifest](https://dist.gocassini.com/manifest.json).
Version-1 exports need `.5`. The stock Go/C API shim is unchanged:
set `Segmentation.Pyannote.Model` to the Nemotron model.

Only the amd64/arm64 C API libraries change from `.5`; ONNX Runtime, C++
wrappers, Go declarations, C header and arm32 binaries are unchanged.
`scripts/build-native.sh` pins the `.6` source.
The Parakeet frontend stays intact. Release checks cover stock Go diarization
with both model precisions, repeated calls and recording/chunk boundaries,
plus the synthetic Parakeet ASR regression on both architectures (arm64
under emulation). CUDA execution is not covered by these CPU packages.

`v1.13.7-cassini.5` rebuilds the amd64/arm64 C API library from
codemyriad/sherpa-onnx commit `5c717ea2b33a724e36602ff5bcd88a72d4db48c6`
(`v1.13.7-cassini`): the `.4` source plus NVIDIA Nemotron-3-Diarization
(k2-fsa/sherpa-onnx#4006), with speaker diarization enabled in the build.
The stock v1.13.7 Go wrapper drives it through the existing
`OfflineSpeakerDiarization` API: pass the Nemotron model as
`Segmentation.Pyannote.Model`; no embedding model or clustering is needed.
Native version: `1.13.7+cassini-parakeet-v3-reference-v1.nemotron-diarization-v1`,
so checks for `+cassini-parakeet-v3-reference-v1` keep working. ONNX Runtime,
the C++ wrapper libraries, the Go source and the C header are unchanged from
`.4`. Both architectures passed Cassini's fp32 synthetic ASR regression and a
four-speaker diarization smoke test before release (arm64 under emulation).

`v1.13.7-cassini.4` aligns the Go source, C header and stock libraries to upstream
v1.13.7. Earlier Cassini tags accidentally used the v1.13.8 declarations.
Only the amd64/arm64 C API and ONNX Runtime binaries are replaced. arm32 remains
stock v1.13.7; the C++ libraries are the upstream v1.13.7 wrappers. Consumers
should pair this module with `github.com/k2-fsa/sherpa-onnx-go v1.13.7`.

The `.4` native source was codemyriad/sherpa-onnx commit
`832bfe50d1e45929e47c9d6e7a65e8a00a855820` (v1.13.7 plus the gated Parakeet v3
frontend), native version `1.13.7+cassini-parakeet-v3-reference-v1`.
The build helper verifies its archive checksum and uses the release's pinned ONNX Runtime assets. The
binaries in .4 are unchanged from .3; the recipe documents how to rebuild that
source and is not a claim of byte-for-byte reproducibility.

From this repository:

```sh
docker buildx build --platform linux/amd64 --build-arg CASSINI_NATIVE_BUILD_JOBS=12 -f scripts/Dockerfile.native --output type=local,dest=/tmp/sherpa-amd64 .
docker buildx build --platform linux/arm64 --build-arg CASSINI_NATIVE_BUILD_JOBS=12 -f scripts/Dockerfile.native --output type=local,dest=/tmp/sherpa-arm64 .
```

Copy only `libsherpa-onnx-c-api.so` and `libonnxruntime.so` into the matching
`lib/*-unknown-linux-gnu` directory. Keep the generated buildinfo with release
records, check each native version and run Cassini's transcription smoke on
both architectures before publishing changed binaries. Never move published
Go tags: publish a new version and update the consumer's go.sum.

`CASSINI-SHA256SUMS` records the released amd64/arm64 libraries. Verify with
`sha256sum -c CASSINI-SHA256SUMS` from the repository root.
