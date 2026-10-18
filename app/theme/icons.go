package theme

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

// The interface icons are pixelarticons (MIT, pixelarticons@2.4.2; docs/design/ui-spec.md, Icons):
// a 24-unit viewBox drawn in 2-unit pixels, a 12 x 12 pixel grid like the avatars. They fill with
// currentColor, so ui.Icon tints them with the color of the surrounding text. Draw them at whole
// multiples of 12 pt (IconSmall, IconRow, IconBar, IconLarge) to keep every pixel on the device grid.
//
//go:embed pixel-icons/*.svg
var pixelIcons embed.FS

// Icon sizes, in points: 1, 1.5, 2 and 3 points a pixel of the grid.
const (
	IconSmall float32 = 12 // meta lines, chevrons, clear buttons
	IconRow   float32 = 18 // sidebar and menu rows, buttons, inspector properties
	IconBar   float32 = 24 // the Settings toolbar
	IconLarge float32 = 36
)

// pixelIcon parses one of the embedded SVGs by its pixelarticons name.
func pixelIcon(name string) *ui.SVG {
	data, err := pixelIcons.ReadFile("pixel-icons/" + name + ".svg")
	if err != nil {
		panic("theme: " + err.Error())
	}
	return ui.MustParseSVG(data)
}

// Icons in the color of the surrounding text.
var (
	IconInbox     = pixelIcon("inbox")
	IconGear      = pixelIcon("gear")
	IconChevron   = pixelIcon("chevron-down")
	IconArrowOut  = pixelIcon("external-link")
	IconCalendar  = pixelIcon("calendar")
	IconCheck     = pixelIcon("check")
	IconCopy      = pixelIcon("copy")
	IconMilestone = pixelIcon("flag")
	IconPlus      = pixelIcon("plus")
	IconSparkle   = pixelIcon("sparkle")
	IconBell      = pixelIcon("bell")
	IconNow       = pixelIcon("circle")
	IconClose     = pixelIcon("close")
	IconSearch    = pixelIcon("search")
	IconMove      = pixelIcon("arrow-right")
	IconClock     = pixelIcon("clock")
	IconTrash     = pixelIcon("trash")
	IconList      = pixelIcon("bulletlist")
	IconBoard     = pixelIcon("grid-3x2")
	IconBack      = pixelIcon("chevron-left")
	IconForward   = pixelIcon("chevron-right")
)

// The sidebar, the inspector's properties and the Settings toolbar.
var (
	IconDone     = pixelIcon("checkbox-on")
	IconStatus   = pixelIcon("invert")
	IconAgent    = pixelIcon("robot")
	IconBranch   = pixelIcon("git-branch")
	IconGeneral  = pixelIcon("sliders")
	IconTerminal = pixelIcon("terminal")
	IconStar     = pixelIcon("sparkle")
	IconOctocat  = pixelIcon("github")
	IconFlag     = IconMilestone
	IconDiamond  = pixelIcon("diamond-gem")
	IconPR       = pixelIcon("git-pull-request")
	IconFile     = pixelIcon("file-text")
)

// ProjectColors tint a project's folder; the color follows the project's name.
var ProjectColors = []ui.Color{
	ui.Hex("#5B5BD6"), ui.Hex("#E0A800"), ui.Hex("#30A46C"), ui.Hex("#E5484D"),
	ui.Hex("#3B82F6"), ui.Hex("#D9488B"), ui.Hex("#0EA5A4"), ui.Hex("#E08A1E"),
}

// ProjectColor is the color of a project's folder in the sidebar.
func ProjectColor(name string) ui.Color {
	var h uint32 = 2166136261
	for _, b := range []byte(name) {
		h = (h ^ uint32(b)) * 16777619
	}
	return ProjectColors[h%uint32(len(ProjectColors))]
}
