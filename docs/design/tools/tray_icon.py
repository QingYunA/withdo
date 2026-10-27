#!/usr/bin/env python3
"""Draws the Windows tray icon: the app icon (a navy square with a check, its short stroke in iris) on the 12 x 12 grid of
the Pixel language, written at 1x to app/theme/sprites/tray.png. The app scales it up whole
(theme.TrayIcon) for the notification area.

    python3 docs/design/tools/tray_icon.py
"""
import os

from PIL import Image

# The three colors of app/resources/icon.png (docs/design/tools/app_icon.py). Both strokes are light,
# so the menu bar's template image cuts the whole check out (theme.TrayTemplateIcon).
COLORS = {"#": (0x1E, 0x29, 0x3B, 0xFF), "o": (0xF8, 0xFA, 0xFC, 0xFF), "i": (0xB1, 0xA9, 0xFF, 0xFF), ".": (0, 0, 0, 0)}

# The square loses one pixel at each corner; the check is two pixels thick and sits in the middle,
# two cells from either side and three from the top. The short stroke is iris, the vertex white.
GRID = [
    ".##########.",
    "############",
    "############",
    "########oo##",
    "#######oo###",
    "######oo####",
    "##ii#oo#####",
    "###ioo######",
    "####o#######",
    "############",
    "############",
    ".##########.",
]


def main():
    img = Image.new("RGBA", (12, 12))
    for y, row in enumerate(GRID):
        assert len(row) == 12, row
        for x, cell in enumerate(row):
            img.putpixel((x, y), COLORS[cell])
    root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    out = os.path.join(root, "app", "theme", "sprites", "tray.png")
    img.save(out)
    print(out)


if __name__ == "__main__":
    main()
