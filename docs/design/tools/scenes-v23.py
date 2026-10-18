"""Dress the day and dusk scenes with Kenney Tiny Town tiles (v23, the mixed version): its grass with
flowers and its path with a grassy edge, two trees at the left and mushrooms at the foot of the lamp post; our fence,
lamp post, sky and hills stay. The tiles are daylight colours, so each scene tints them to its own
grass. The night (the week) is left as it is: its stacks stand across the whole ground.
    python3 docs/design/tools/scenes-v23.py
reads docs/design/tools/scene-base/{day,dusk}.png and third_party/kenney-tiny-town/Tiles, writes
app/theme/sprites/scenes/{day,dusk}.png.
"""
import random
from PIL import Image

TILES = 'third_party/kenney-tiny-town/Tiles/tile_{:04d}.png'
GRASS, GRASS_TUFT, GRASS_FLOWER, PATH_EDGE, TREE, TREE_SMALL, MUSHROOMS = 0, 1, 2, 13, 16, 28, 29
FENCE_X = range(166, 246)  # our fence, kept over the new grass
LAMP_X = range(244, 262)   # our lamp post, kept over the new ground

def tile(i, k):
    im = Image.open(TILES.format(i)).convert('RGBA')
    px = im.load()
    for y in range(16):
        for x in range(16):
            r, g, b, a = px[x, y]
            if a:
                px[x, y] = (min(255, round(r * k[0])), min(255, round(g * k[1])), min(255, round(b * k[2])), a)
    return im

def dress(name):
    base = Image.open(f'docs/design/tools/scene-base/{name}.png').convert('RGBA')
    ours = base.getpixel((10, 60))
    theirs = Image.open(TILES.format(GRASS)).convert('RGBA').getpixel((8, 8))
    k = [ours[i] / theirs[i] for i in range(3)]
    out = base.copy()
    rnd = random.Random(3)
    for x in range(0, base.width, 16):
        out.alpha_composite(tile(rnd.choice([GRASS, GRASS, GRASS_TUFT, GRASS, GRASS_FLOWER, GRASS_TUFT]), k).crop((0, 0, 16, 8)), (x, 56))
        out.alpha_composite(tile(PATH_EDGE, k), (x, 64))
    src, dst = base.load(), out.load()
    for y in range(56, base.height):
        for x in FENCE_X:
            r, g, b, _ = src[x, y]
            if y < 64 and r > g:  # the fence's rails and posts are warm, the grass green
                dst[x, y] = src[x, y]
        for x in LAMP_X:
            r, g, b, _ = src[x, y]
            if r < 70 and g < 70 and b < 100:  # the post
                dst[x, y] = src[x, y]
    out.alpha_composite(tile(TREE, k), (4, 42))
    out.alpha_composite(tile(TREE_SMALL, k), (13, 46))
    out.alpha_composite(tile(MUSHROOMS, k), (243, 52))  # at the foot of the lamp post
    out.save(f'app/theme/sprites/scenes/{name}.png')

for n in ('day', 'dusk'):
    dress(n)
