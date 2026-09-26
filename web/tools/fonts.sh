#!/usr/bin/env bash
# Сборка веб-шрифтов: TTF из ios/Shtil/Resources/Fonts → урезанный WOFF2 (латиница, кириллица, знаки, стрелки).
# Нужны fonttools и brotli (только для сборки, в приложение они не входят):
#   python3 -m venv /tmp/fv && /tmp/fv/bin/pip install fonttools brotli && PY=/tmp/fv/bin/python web/tools/fonts.sh
set -euo pipefail
PY=${PY:-python3}
SRC="$(dirname "$0")/../../ios/Shtil/Resources/Fonts"
OUT="$(dirname "$0")/../fonts"
UNICODES="U+0020-007E,U+00A0-00FF,U+0131,U+0152-0153,U+02C6,U+02DA,U+02DC,U+0400-04FF,U+2010-2015,U+2018-201E,U+2020-2022,U+2026,U+2030,U+2039-203A,U+20AC,U+20BD,U+2116,U+2122,U+2190-2193,U+2212,U+2215,U+2713,U+00D7,U+2197,U+21BB"
for f in Onest GolosText Spectral-Medium Spectral-SemiBold Spectral-Italic IBMPlexMono-Regular IBMPlexMono-Medium; do
  "$PY" -m fontTools.subset "$SRC/$f.ttf" --unicodes="$UNICODES" --layout-features='*' --flavor=woff2 --output-file="$OUT/$f.woff2"
done
ls -l "$OUT"/*.woff2 | awk '{s+=$5; print $5, $9} END {print s, "итого"}'
