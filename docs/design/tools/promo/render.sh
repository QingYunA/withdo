#!/usr/bin/env bash
# Render the promo video (docs/design/promo.md) in one language.
#   docs/design/tools/promo/render.sh zh|en
# Writes out/promo/adhd-todo-summary-<lang>-4k.mp4 and -1080p.mp4, and stills in out/promo/stills-<lang>/.
# The frames and the sound come from the Go tests behind the `promo` build tag (app/views/promo_*_test.go);
# ffmpeg puts the cards, the window segment and the sound together.
set -euo pipefail

lang="${1:-}"
case "$lang" in zh | en) ;; *) echo "usage: $0 zh|en" >&2; exit 2 ;; esac

cd "$(git rev-parse --show-toplevel)"
root="$PWD"
ffmpeg="$(command -v ffmpeg || echo /opt/homebrew/bin/ffmpeg)"
work="$root/out/promo/work-$lang"
out="$root/out/promo"
stills="$out/stills-$lang"
rm -rf "$work" "$stills"
mkdir -p "$work" "$stills"

started=$SECONDS
echo "== frames, cards and sound ($lang)"
PROMO_LANG="$lang" PROMO_OUT="$work" go test -tags promo ./app/views -run 'TestPromo(Window|Cards)' -count=1 -timeout 30m

# timing.env (written by the window test): when the window segment starts and ends and how long the
# video is, in seconds. The title card runs to WIN_START + 0.2 and the end card, 5.1 s long, starts
# 0.2 s before the window ends; both cuts are 200 ms crossfades.
# shellcheck disable=SC1091
. "$work/timing.env"
end_at="$(awk -v e="$WIN_END" 'BEGIN { printf "%.3f", e - 0.2 }')"
fade_at="$(awk -v t="$TOTAL" 'BEGIN { printf "%.3f", t - 0.5 }')"

echo "== encode ($lang, $TOTAL s)"
"$ffmpeg" -hide_banner -loglevel warning -nostats -y \
  -framerate 60 -i "$work/t%04d.png" \
  -framerate 60 -i "$work/w%05d.png" \
  -loop 1 -framerate 60 -t 5.1 -i "$work/end.png" \
  -i "$work/audio.wav" \
  -filter_complex "
    [0:v]format=gbrp,fade=t=in:st=0:d=0.3[title];
    [1:v]format=gbrp[win];
    [2:v]format=gbrp,fade=t=out:st=4.8:d=0.3[end];
    [title][win]xfade=transition=fade:duration=0.2:offset=$WIN_START[a];
    [a][end]xfade=transition=fade:duration=0.2:offset=$end_at,split[v4][v1];
    [v4]scale=out_color_matrix=bt709:out_range=tv,format=yuv420p[k4];
    [v1]scale=1920:1080:flags=area:out_color_matrix=bt709:out_range=tv,format=yuv420p[k1];
    [3:a]afade=t=out:st=$fade_at:d=0.5,aformat=channel_layouts=stereo,asplit[au4][au1]" \
  -map '[k4]' -map '[au4]' -t "$TOTAL" -r 60 -c:v libx264 -preset medium -crf 14 -maxrate 80M -bufsize 160M \
    -colorspace bt709 -color_primaries bt709 -color_trc bt709 -movflags +faststart \
    -c:a aac -b:a 160k "$out/adhd-todo-summary-$lang-4k.mp4" \
  -map '[k1]' -map '[au1]' -t "$TOTAL" -r 60 -c:v libx264 -preset slow -crf 14 -maxrate 20M -bufsize 40M \
    -colorspace bt709 -color_primaries bt709 -color_trc bt709 -movflags +faststart \
    -c:a aac -b:a 160k "$out/adhd-todo-summary-$lang-1080p.mp4"

echo "== stills ($lang)"
# stills.txt (written by the window test): a name and a video second per line
while read -r name at; do
  "$ffmpeg" -nostdin -hide_banner -loglevel error -y -ss "$at" -i "$out/adhd-todo-summary-$lang-1080p.mp4" -frames:v 1 "$stills/$name.png"
done < "$work/stills.txt"

rm -rf "$work"
echo "done in $((SECONDS - started)) s: $out/adhd-todo-summary-$lang-{4k,1080p}.mp4"
