package theme

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

// googleG is Google's four-color "G" mark, the one logo in the app that is not pixel art: Google's
// branding rules for "Continue with Google" forbid redrawing or recoloring it. Draw it with
// ui.Image, which keeps the SVG's own colors.
//
//go:embed google-g.svg
var googleG []byte

// IconGoogle is the G mark as an SVG; its viewBox is square.
var IconGoogle = ui.MustParseSVG(googleG)
