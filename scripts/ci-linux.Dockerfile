# The Linux environment of the `check` job in .github/workflows/ci.yml, for scripts/check.sh linux.
# GO_VERSION comes from go.mod (check.sh passes it).
ARG GO_VERSION=1.27.1
FROM golang:${GO_VERSION}-bookworm
# mygo loads pango at run time; the Chinese tests measure Han glyphs and need a CJK font.
RUN apt-get update && apt-get install -y --no-install-recommends libpangocairo-1.0-0 fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*
