#!/usr/bin/env python3
"""Draws the app icon (v40 C, docs/design/mockups/v40.html): a pixel check on the navy square, the
short stroke in iris (you decide what to do) and the long stroke in white (the agent gets it done),
on a 24 x 24 grid with a brush three cells square. Writes

- app/resources/icon.png, 1024 px: the square keeps the shape of the file it replaces (its alpha), the
  check is drawn 28 px a cell in the middle of the square;
- android/app/src/main/res/drawable/ic_launcher_foreground.xml: the same check, 3 dp a cell, in the
  middle of the 108 dp adaptive canvas (inside its 66 dp safe zone).

    python3 docs/design/tools/app_icon.py
"""
import os

from PIL import Image

NAVY = (0x1E, 0x29, 0x3B, 0xFF)
COLORS = {"i": "#B1A9FF", "o": "#F8FAFC"}  # iris 11 (dark), the white of the old check

GRID, BRUSH, SHORT, LONG = 24, 3, 5, 11


def check():
    """The cells of the check as {(x, y): key}, its box starting at (0, 0). The vertex belongs to the
    long stroke."""
    cells = {}

    def square(x, y, key):
        for dy in range(BRUSH):
            for dx in range(BRUSH):
                cells[(x + dx, y + dy)] = key

    vy = LONG
    for s in range(SHORT):
        square(s, vy - SHORT + s, "i")
    for s in range(LONG + 1):
        square(SHORT + s, vy - s, "o")
    return cells


def box(cells):
    return max(x for x, _ in cells) + 1, max(y for _, y in cells) + 1


def rgba(hex_color):
    return tuple(int(hex_color[i : i + 2], 16) for i in (1, 3, 5)) + (0xFF,)


def write_png(root, cells):
    path = os.path.join(root, "app", "resources", "icon.png")
    shape = Image.open(path).convert("RGBA").getchannel("A")
    left, top, right, bottom = shape.getbbox()
    img = Image.new("RGBA", shape.size, NAVY)
    img.putalpha(shape)
    cell = 28
    w, h = box(cells)
    x0 = left + (right - left - w * cell) // 2
    y0 = top + (bottom - top - h * cell) // 2
    for (x, y), key in cells.items():
        img.paste(rgba(COLORS[key]), (x0 + x * cell, y0 + y * cell, x0 + (x + 1) * cell, y0 + (y + 1) * cell))
    img.save(path)
    return path


def write_android(root, cells):
    path = os.path.join(root, "android", "app", "src", "main", "res", "drawable", "ic_launcher_foreground.xml")
    dp = 3
    w, h = box(cells)
    x0, y0 = (108 - w * dp) / 2, (108 - h * dp) / 2
    paths = []
    for key in ("i", "o"):
        data = "".join(
            f"M{x0 + x * dp:g},{y0 + y * dp:g}h{dp}v{dp}h-{dp}z"
            for (x, y), k in sorted(cells.items(), key=lambda c: (c[0][1], c[0][0]))
            if k == key
        )
        paths.append(f'    <path\n        android:fillColor="{COLORS[key]}"\n        android:pathData="{data}" />')
    xml = (
        '<?xml version="1.0" encoding="utf-8"?>\n'
        "<!-- The desktop icon's check (docs/design/tools/app_icon.py): 24 x 24 grid, 3 dp a cell,\n"
        "     centered in the 108 dp adaptive canvas (inside its 66 dp safe zone). -->\n"
        '<vector xmlns:android="http://schemas.android.com/apk/res/android"\n'
        '    android:width="108dp"\n'
        '    android:height="108dp"\n'
        '    android:viewportWidth="108"\n'
        '    android:viewportHeight="108">\n' + "\n".join(paths) + "\n</vector>\n"
    )
    with open(path, "w") as f:
        f.write(xml)
    return path


def main():
    root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
    cells = check()
    assert box(cells) == (SHORT + LONG + BRUSH, LONG + BRUSH) and max(box(cells)) <= GRID
    print(write_png(root, cells))
    print(write_android(root, cells))


if __name__ == "__main__":
    main()
