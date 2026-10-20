package theme

import (
	"strings"
	"testing"

	"github.com/QingYunA/adhd-todo/app/data"
	"github.com/egoist/mygo/ui"
)

func TestEveryVendoredPixelIconParsesOnTheTwentyFourUnitGrid(t *testing.T) {
	files, err := pixelIcons.ReadDir("pixel-icons")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no icons embedded")
	}
	for _, f := range files {
		b, err := pixelIcons.ReadFile("pixel-icons/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if _, err := ui.ParseSVG(b); err != nil {
			t.Errorf("%s does not parse: %v", f.Name(), err)
		}
		if !strings.Contains(s, `fill="currentColor"`) || !strings.Contains(s, `viewBox="0 0 24 24"`) {
			t.Errorf("%s is not a currentColor icon on the 24 unit grid", f.Name())
		}
		if strings.Contains(s, ` width=`) || strings.Contains(s, ` height=`) {
			t.Errorf("%s still has a width or a height", f.Name())
		}
	}
}

func TestEveryIconVariableIsSet(t *testing.T) {
	all := map[string]*ui.SVG{
		"Inbox": IconInbox, "Gear": IconGear, "Chevron": IconChevron, "ArrowOut": IconArrowOut, "Calendar": IconCalendar,
		"Check": IconCheck, "Copy": IconCopy, "Milestone": IconMilestone, "Plus": IconPlus, "Sparkle": IconSparkle,
		"Bell": IconBell, "Now": IconNow, "Close": IconClose, "Search": IconSearch, "Move": IconMove, "Clock": IconClock,
		"Trash": IconTrash, "List": IconList, "Board": IconBoard, "Back": IconBack, "Forward": IconForward,
		"Done": IconDone, "Box": IconBox, "Status": IconStatus, "Agent": IconAgent, "Branch": IconBranch, "General": IconGeneral,
		"Terminal": IconTerminal, "Star": IconStar, "Octocat": IconOctocat, "Flag": IconFlag, "Diamond": IconDiamond,
		"PR": IconPR, "Send": IconSend, "Agents": IconAgents, "GitHub": IconGitHub, "Folder": IconFolder,
		"Cloud": IconCloud, "Laptop": IconLaptop, "Phone": IconPhone,
	}
	for name, s := range all {
		if s == nil {
			t.Errorf("Icon%s is nil", name)
		}
	}
}

func TestIconSizesAreWholeMultiplesOfTheTwelvePixelGrid(t *testing.T) {
	for _, size := range []float32{IconSmall, IconRow, IconBar, IconLarge, IconProp, IconMenu, GlyphSize} {
		switch size {
		case 12, 18, 24, 36:
		default:
			t.Errorf("%v pt is not 12, 18, 24 or 36", size)
		}
	}
}

func TestStatusGlyphsAreTwelveSquarePixelDrawings(t *testing.T) {
	for _, s := range data.Statuses {
		rows, ok := glyphSprites[s]
		if !ok || len(rows) != 12 {
			t.Fatalf("%s: %d rows", s, len(rows))
		}
		for y, r := range rows {
			if len(r) != 12 {
				t.Errorf("%s row %d: %d pixels", s, y, len(r))
			}
		}
	}
	count := func(rows []string, ch string) int { return strings.Count(strings.Join(rows, ""), ch) }
	if count(glyphSprites[data.Done], "w") == 0 || count(glyphSprites[data.Blocked], "w") == 0 {
		t.Error("the check and the bang are white on the disc")
	}
	if count(glyphSprites[data.Backlog], "c") >= count(glyphSprites[data.Todo], "c") {
		t.Error("the backlog ring is dashed: it has fewer pixels than the todo ring")
	}
	if count(glyphDiscOnly, "w") != 0 {
		t.Error("the completing disc has no check yet")
	}
}
