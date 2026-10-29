package theme

import (
	"embed"

	"github.com/QingYunA/adhd-todo/internal/domain"
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
	IconImage     = pixelIcon("image")
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

// The category icon editor's tools; its up and right arrows are IconSend and IconMove.
var (
	IconArrowLeft = pixelIcon("arrow-left")
	IconArrowDown = pixelIcon("arrow-down")
	IconMinus     = pixelIcon("minus")
	IconFlip      = pixelIcon("flip-horizontal-2")
)

// The sidebar, the inspector's properties and the Settings sidebar.
var (
	IconDone     = pixelIcon("checkbox-on")
	IconBox      = pixelIcon("checkbox")
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
	IconCloud    = pixelIcon("cloud")
	IconLaptop   = pixelIcon("laptop")
	IconPhone    = pixelIcon("smartphone")
	IconUser     = pixelIcon("user")
	IconMail     = pixelIcon("mail")
	IconSummary  = pixelIcon("chart")
)

// ProjectColors tint a project's folder; the color follows the project's name.
var ProjectColors = []ui.Color{
	ui.Hex("#5B5BD6"), ui.Hex("#FFC53D"), ui.Hex("#30A46C"), ui.Hex("#E5484D"),
	ui.Hex("#0090FF"), ui.Hex("#D6409F"), ui.Hex("#12A594"), ui.Hex("#F76B15"),
}

// projectInk is step 11 of each project color's Radix scale (iris, amber, green, red, blue, pink,
// teal, orange), for the light appearance then the dark one: the color of a word, where
// ProjectColors (step 9) is for icons.
var projectInk = [2][8]ui.Color{
	{ui.Hex("#5753C6"), ui.Hex("#AB6400"), ui.Hex("#218358"), ui.Hex("#CE2C31"),
		ui.Hex("#0D74CE"), ui.Hex("#C2298A"), ui.Hex("#008573"), ui.Hex("#CC4E00")},
	{ui.Hex("#B1A9FF"), ui.Hex("#FFCA16"), ui.Hex("#3DD68C"), ui.Hex("#FF9592"),
		ui.Hex("#70B8FF"), ui.Hex("#FF8DCC"), ui.Hex("#0BD8B6"), ui.Hex("#FFA057")},
}

// CategoryInk is a category's color as a word: step 11 of its scale, for the appearance of t.
func (t *Tokens) CategoryInk(key string) ui.Color {
	i := 0
	if t.Dark {
		i = 1
	}
	for j, k := range domain.CategoryColors {
		if k == key {
			return projectInk[i][j]
		}
	}
	return projectInk[i][0]
}

// CategoryColor is the color a category's key stands for: domain.CategoryColors are the project
// colors by another name, in the same order. A key this build does not know gets the first color.
func CategoryColor(key string) ui.Color {
	for i, k := range domain.CategoryColors {
		if k == key {
			return ProjectColors[i]
		}
	}
	return ProjectColors[0]
}

// ProjectColor is the color of a project's folder in the sidebar.
func ProjectColor(name string) ui.Color {
	var h uint32 = 2166136261
	for _, b := range []byte(name) {
		h = (h ^ uint32(b)) * 16777619
	}
	return ProjectColors[h%uint32(len(ProjectColors))]
}
