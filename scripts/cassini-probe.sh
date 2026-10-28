#!/usr/bin/env bash
# Probe the libraries in this checkout's lib/ through this module's Go wrapper:
# runtime version, the -60 dB quiet-audio probe with Cassini's production
# Parakeet v3 int8 and Parakeet Ultra int8 bundles (Ultra keeps its plain
# https://huggingface.co/moondream/parakeet-ultra encoder url), and Sortformer
# diarization with the production Nemotron-3-Diarization int8 model (ONNX
# interface version 2) through Segmentation.Sortformer.
# Usage: scripts/cassini-probe.sh MODELS_DIR OUTPUT_DIR
# Needs: go, curl, zstd, sha256sum. Inputs are pinned by sha256.
set -euo pipefail
[[ $# -eq 2 ]] || { echo 'usage: cassini-probe.sh MODELS_DIR OUTPUT_DIR' >&2; exit 2; }
root=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$1" "$2"
models=$(cd "$1" && pwd); out=$(cd "$2" && pwd)

# fetch URL DEST SHA256
fetch() {
  if [[ -f $2 ]] && echo "$3  $2" | sha256sum -c --quiet - 2>/dev/null; then return; fi
  mkdir -p "$(dirname "$2")"
  curl --fail --location --retry 3 --silent --show-error -o "$2.tmp" "$1"
  echo "$3  $2.tmp" | sha256sum -c --quiet -
  mv "$2.tmp" "$2"
}
# dist DEST ARTIFACT_SHA256 UNCOMPRESSED_SHA256: a dist.gocassini.com
# seekable-zstd model file, addressed by its compressed sha256.
dist() {
  if [[ -f $1 ]] && echo "$3  $1" | sha256sum -c --quiet - 2>/dev/null; then return; fi
  fetch "https://dist.gocassini.com/models/files/$2/$(basename "$1").zst" "$1.zst" "$2"
  zstd -q -d -f "$1.zst" -o "$1"
  rm -f "$1.zst"
  echo "$3  $1" | sha256sum -c --quiet -
}

# Public English sample (3.85 s, 24 kHz), from the k2-fsa Parakeet v3 int8 repo.
fetch https://huggingface.co/csukuangfj/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8/resolve/2bda32ec70b097a55adaa07d9a7173915b43cc78/test_wavs/en.wav \
  "$models/en.wav" 148b936b43ce7c546a866e64da059f0458aee2d65e617f16e9d94f06e8d99ed6
# Public four-speaker sample (56.9 s, 16 kHz) from k2-fsa's diarization tests.
fetch https://github.com/k2-fsa/sherpa-onnx/releases/download/speaker-segmentation-models/0-four-speakers-zh.wav \
  "$models/0-four-speakers-zh.wav" bedf036caed208386c67b4ef4b11f83d74dd0d420b102163a1c33cd09cde7010

# dist.gocassini.com manifest: parakeet-tdt-0.6b-v3-int8.
v3=$models/parakeet-tdt-0.6b-v3-int8
dist "$v3/encoder.int8.onnx" cb9e98257d3bc2611b43d9897e67454e5893aff1ffc90229ca3fa7f9ed5bb22e acfc2b4456377e15d04f0243af540b7fe7c992f8d898d751cf134c3a55fd2247
dist "$v3/decoder.int8.onnx" b14c3967107f0a5e4e9c0bde770c204202bde8d1a6255c27a863930e8c887c3d 179e50c43d1a9de79c8a24149a2f9bac6eb5981823f2a2ed88d655b24248db4e
dist "$v3/joiner.int8.onnx" 1ef3c78137d39d45cfad961be5a45bf7abd3c13fa721c7da2327b0eb3d0e8ce6 3164c13fc2821009440d20fcb5fdc78bff28b4db2f8d0f0b329101719c0948b3
dist "$v3/tokens.txt" 5b3811f0c23fefc86835981f0a23ecfb817999df272ad744c597e5b669ea1b4a d58544679ea4bc6ac563d1f545eb7d474bd6cfa467f0a6e2c1dc1c7d37e3c35d
# parakeet-ultra-0.6b-int8: encoder url metadata is the plain
# https://huggingface.co/moondream/parakeet-ultra, which stock v2.0.1 rejects.
ultra=$models/parakeet-ultra-0.6b-int8
dist "$ultra/encoder.int8.onnx" caf7a38f2ad3224c1863564fb6e975a11aa3ae6ad1e7ad0bee5c4f5e018babb4 299617fefd4664d1c115b691dcb6413cabcdbcc694cdf3cb4464b72c5fc3ae69
dist "$ultra/decoder.int8.onnx" 69e2e91d01225bc552f8b0113630562fc52622cb2a3c61ef4445ee779f06b295 1fab98fe6c12aded87d2da66272cc9e148d0a0044ce3850a12fe56302ec4a922
dist "$ultra/joiner.int8.onnx" 639fda62a71242a8d7cefb6647f2094eb8ac783925ce91cbd047a0de6988f396 8a71aaccfdba3d451775507a889afd2257d2e754463fd81095a8c8f290f924c2
dist "$ultra/tokens.txt" 5b3811f0c23fefc86835981f0a23ecfb817999df272ad744c597e5b669ea1b4a d58544679ea4bc6ac563d1f545eb7d474bd6cfa467f0a6e2c1dc1c7d37e3c35d
grep -aqF 'https://huggingface.co/moondream/parakeet-ultra' "$ultra/encoder.int8.onnx"
if grep -aq 'moondream/parakeet-ultra (' "$ultra/encoder.int8.onnx"; then
  echo 'Ultra encoder url is annotated; the probe needs the plain url' >&2; exit 1
fi
# nemotron-3-diarization-int8 (interface version 2).
diar=$models/nemotron-3-diarization-int8/model.int8.onnx
dist "$diar" 794720a59892189484d38a186d2b350018b4194cae55292f501b2849d6313290 47c221ea9b4d4e7f6c108bd098e769bc706cdc986c29f6133c93cae335fe6779

(cd "$root/scripts/cassini-probe" && go build -buildvcs=false -o "$out/cassini-probe" .)
"$out/cassini-probe" -wav "$models/en.wav" -asr "$v3" -asr "$ultra" \
  -sortformer "$diar" -diar-wav "$models/0-four-speakers-zh.wav" -min-speakers 4 \
  2>&1 | tee "$out/probe.log"
grep -qx PASS "$out/probe.log"
