#!/usr/bin/env bash
# Render the promo video (docs/design/promo.md) in one language.
#   docs/design/tools/promo/render.sh zh|en
# Writes out/promo/adhd-todo-summary-<lang>-4k.mp4 and -1080p.mp4, and stills in out/promo/stills-<lang>/.
# The frames and the sound come from the Go tests behind the `promo` build tag (app/views/promo_*_test.go);
# ffmpeg puts cards, window, backdrop and sound together.
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

# Video seconds: the title card runs to 2.6 s and the window from 2.4 s to 23.1 s (200 ms crossfades
# centred on 2.5 s and 23 s); the end card starts at 22.9 s and runs to 28 s.
echo "== encode ($lang)"
"$ffmpeg" -hide_banner -loglevel warning -nostats -y \
  -framerate 60 -i "$work/t%04d.png" \
  -framerate 60 -i "$work/w%05d.png" \
  -loop 1 -framerate 60 -i "$work/backdrop-window.png" \
  -loop 1 -framerate 60 -t 5.1 -i "$work/end.png" \
  -i "$work/audio.wav" \
  -filter_complex "
    [0:v]format=gbrp,fade=t=in:st=0:d=0.3[title];
    [2:v][1:v]overlay=820:360:shortest=1:format=gbrp[win];
    [3:v]format=gbrp,fade=t=out:st=4.8:d=0.3[end];
    [title][win]xfade=transition=fade:duration=0.2:offset=2.4[a];
    [a][end]xfade=transition=fade:duration=0.2:offset=22.9,split[v4][v1];
    [v4]scale=out_color_matrix=bt709:out_range=tv,format=yuv420p[k4];
    [v1]scale=1920:1080:flags=area:out_color_matrix=bt709:out_range=tv,format=yuv420p[k1];
    [4:a]afade=t=out:st=27.5:d=0.5,aformat=channel_layouts=stereo[a4];
    [a4]asplit[au4][au1]" \
  -map '[k4]' -map '[au4]' -t 28 -r 60 -c:v libx264 -preset medium -crf 14 -maxrate 80M -bufsize 160M \
    -colorspace bt709 -color_primaries bt709 -color_trc bt709 -movflags +faststart \
    -c:a aac -b:a 160k "$out/adhd-todo-summary-$lang-4k.mp4" \
  -map '[k1]' -map '[au1]' -t 28 -r 60 -c:v libx264 -preset slow -crf 14 -maxrate 20M -bufsize 40M \
    -colorspace bt709 -color_primaries bt709 -color_trc bt709 -movflags +faststart \
    -c:a aac -b:a 160k "$out/adhd-todo-summary-$lang-1080p.mp4"

echo "== stills ($lang)"
still() { # name seconds
  "$ffmpeg" -hide_banner -loglevel error -y -ss "$2" -i "$out/adhd-todo-summary-$lang-1080p.mp4" -frames:v 1 "$stills/$1.png"
}
still 01-title-hop 1.2
still 02-crossfade 2.5
still 03-pointer-at-summary 3.45
still 04-scene-throws 7.0
still 05-scene-hop 11.8
still 06-paragraph 13.0
still 07-ledger 15.2
still 08-pointer-at-week 16.3
still 09-week-falling 19.0
still 10-week-end 22.0
still 11-end-card 26.0

echo "done in $((SECONDS - started)) s: $out/adhd-todo-summary-$lang-{4k,1080p}.mp4"
