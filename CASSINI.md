# Cassini release maintenance

`v1.13.7-cassini.4` aligns the Go source, C header and stock libraries to upstream
v1.13.7. Earlier Cassini tags accidentally used the v1.13.8 declarations.
Only the amd64/arm64 C API and ONNX Runtime binaries are replaced. arm32 remains
stock v1.13.7; the C++ libraries are the upstream v1.13.7 wrappers. Consumers
should pair this module with `github.com/k2-fsa/sherpa-onnx-go v1.13.7`.

The patched native source is codemyriad/sherpa-onnx commit
`832bfe50d1e45929e47c9d6e7a65e8a00a855820` (v1.13.7 plus the gated Parakeet v3
frontend). The build helper verifies its archive checksum and uses the release's
pinned ONNX Runtime assets. Native version: `1.13.7+cassini-parakeet-v3-reference-v1`.
The binaries in .4 are unchanged from .3; the new recipe documents how to
rebuild that source and is not a claim of byte-for-byte reproducibility.

From this repository:

```sh
docker buildx build --platform linux/amd64 -f scripts/Dockerfile.native --output type=local,dest=/tmp/sherpa-amd64 .
docker buildx build --platform linux/arm64 -f scripts/Dockerfile.native --output type=local,dest=/tmp/sherpa-arm64 .
```

Copy only `libsherpa-onnx-c-api.so` and `libonnxruntime.so` into the matching
`lib/*-unknown-linux-gnu` directory. Keep the generated buildinfo with release
records, check each native version and run Cassini's transcription smoke on
both architectures before publishing changed binaries. Never move published
Go tags: publish a new version and update the consumer's go.sum.

`CASSINI-SHA256SUMS` records the released amd64/arm64 libraries. Verify with
`sha256sum -c CASSINI-SHA256SUMS` from the repository root.
