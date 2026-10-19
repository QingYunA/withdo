"""Draw the v22 actors (24 x 24) the way actors/codex is drawn: an outline, a lit top-left edge, a
shaded bottom-right edge and a translucent shadow one cell down-right. Each actor grows out of its
official mark (app/theme/harness_marks.go): Cursor is the cube with its facet, not a mouse pointer.
    python3 docs/design/tools/actors-v22.py app/theme/sprites/actors
"""
import math, os, sys
from PIL import Image

N = 24
SHADOW = (0x28, 0x10, 0x19, 82)

def hexc(h, a=255):
    h = h.lstrip('#'); return (int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16), a)

def grid(rows):
    """ASCII rows (<=24 wide) placed bottom-aligned: '.' empty, '#' body, other chars details."""
    rows = [r.rstrip() for r in rows]
    g = [['.'] * N for _ in range(N)]
    top = N - 1 - len(rows)  # leave the last row for the shadow
    w = max(len(r) for r in rows)
    left = (N - w) // 2
    for y, r in enumerate(rows):
        for x, ch in enumerate(r):
            g[top + y][left + x] = ch
    return g

def shift(g, dx, dy):
    o = [['.'] * N for _ in range(N)]
    for y in range(N):
        for x in range(N):
            if g[y][x] != '.' and 0 <= x + dx < N and 0 <= y + dy < N:
                o[y + dy][x + dx] = g[y][x]
    return o

def render(g, pal, details, outline, base_at=None, ground=None):
    """pal: dict b,c,d,f (highlight, light, base, shade); base_at(x,y) may return another pal."""
    filled = lambda x, y: 0 <= x < N and 0 <= y < N and g[y][x] != '.' and g[y][x] != 'o'
    im = Image.new('RGBA', (N, N), (0, 0, 0, 0))
    px = im.load()
    # outline: explicit 'o' plus every empty cell next to the body (4-neighbours)
    for y in range(N):
        for x in range(N):
            if g[y][x] == 'o':
                px[x, y] = outline
            elif g[y][x] == '.' and any(filled(x + a, y + b) for a, b in ((1, 0), (-1, 0), (0, 1), (0, -1))):
                px[x, y] = outline
    isout = lambda x, y: not (0 <= x < N and 0 <= y < N) or px[x, y][3] == 255 and g[y][x] in '.o'
    for y in range(N):
        for x in range(N):
            ch = g[y][x]
            if ch in '.o':
                continue
            if ch != '#':
                px[x, y] = details[ch]
                continue
            p = base_at(x, y) if base_at else pal
            up, lf, dn, rt = isout(x, y - 1), isout(x - 1, y), isout(x, y + 1), isout(x + 1, y)
            px[x, y] = p['b'] if up and lf else p['c'] if up or lf else p['f'] if dn or rt else p['d']
    if ground:  # a floating actor's shadow on the grass
        cx, w = ground
        for x in range(cx - w, cx + w + 1):
            if px[x, N - 2][3] == 0:
                px[x, N - 2] = SHADOW
    snap = im.copy()
    for y in range(N):
        for x in range(N):
            if snap.getpixel((x, y))[3] == 255 and x + 1 < N and y + 1 < N and px[x + 1, y + 1][3] == 0:
                px[x + 1, y + 1] = SHADOW
    return im

def star(sx=1.0, sy=1.0, rot=0.0, r=8.6, k=2.3):
    g = [['.'] * N for _ in range(N)]
    cx, cy = 11.5, 12.0
    for y in range(N):
        for x in range(N):
            dx, dy = (x - cx) / sx, (y - cy) / sy
            if rot:
                c, s = math.cos(rot), math.sin(rot)
                dx, dy = dx * c - dy * s, dx * s + dy * c
            h = lambda d: r * max(0.0, 1 - abs(d) / r) ** k
            if (abs(dx) <= h(dy) + 0.5 and abs(dy) <= r) or (abs(dy) <= h(dx) + 0.5 and abs(dx) <= r):
                g[y][x] = '#'
    return g
def face(g, eyes, y, mouth=None):
    for (x, ch) in eyes:
        g[y][x] = ch; g[y + 1][x] = ch
    if mouth:
        for (x, yy) in mouth:
            g[yy][x] = 'm'
    return g
def place(rows, left=None, bottom=22):
    g = [['.'] * N for _ in range(N)]
    w = max(len(r) for r in rows)
    left = (N - w) // 2 if left is None else left
    top = bottom - len(rows) + 1
    for y, r in enumerate(rows):
        for x, ch in enumerate(r):
            if ch != '.':
                g[top + y][left + x] = ch
    return g

def paint(g, pts, ch):
    for x, y in pts:
        g[y][x] = ch
    return g

def find(g, ch):
    return [(x, y) for y in range(N) for x in range(N) if g[y][x] == ch]

# ---------- Cursor: the mark's hexagonal cube, its triangular facet the face ----------
CUBE = [
    '........T........',
    '......TTTTT......',
    '....TTTTTTTTT....',
    '..TTTTTTTTTTTTT..',
    'TTTTTTTTTTTTTTTTT',
    'lFFFFFFFFFFFFFFFr',
    'llFFFFFFFFFFFFFFr',
    'lllFFFFFFFFFFFFrr',
    'llllFFFFFFFFFFFrr',
    'lllllFFFFFFFFFrrr',
    'llllllFFFFFFFFrrr',
    'lllllllFFFFFFrrrr',
    'llllllllFFFFFrrrr',
    'llllllllFFFFrrrrr',
    'llllllllFFFrrrrrr',
    '..lllllllFFrrrr..',
    '....llllllFrr....',
    '......lllllrr....',
    '........lr.......',
]
CUR_D = {'T': hexc('6a6a76'), 'l': hexc('3b3b44'), 'r': hexc('26262c'), 'F': hexc('ececf2'), 'k': hexc('1b1b20'),
         'W': hexc('ffffff'), 'h': hexc('1b1b20'), 's': hexc('ffffff'), 'E': hexc('8a8a98')}
CUR_O = hexc('0c0c10')
def cube(eyes='open', bottom=20, dx=0, flash=False):
    g = place(CUBE, bottom=bottom)
    g = shift(g, dx, 0)
    fx = [p for p in find(g, 'F')]
    top = min(y for x, y in fx)
    if flash:
        paint(g, fx, 'W')
    # a darker edge where the facet meets the cap, as the mark's line
    paint(g, [(x, y) for x, y in find(g, 'T') if y == top - 1 and (x + y) % 1 == 0], 'T')
    y0 = top + 2
    row = [x for x, y in find(g, 'F') + find(g, 'W') if y == y0]
    c = (min(row) + max(row)) // 2
    ex = [c - 3, c + 2]  # the eyes, on the facet's wide top
    if eyes == 'open':
        paint(g, [(ex[0], y0), (ex[0], y0 + 1), (ex[1], y0), (ex[1], y0 + 1)], 'k')
    elif eyes == 'blink':
        paint(g, [(ex[0], y0 + 1), (ex[1], y0 + 1)], 'k')
    elif eyes == 'happy':
        paint(g, [(ex[0], y0), (ex[0] - 1, y0 + 1), (ex[0] + 1, y0 + 1), (ex[1], y0), (ex[1] - 1, y0 + 1), (ex[1] + 1, y0 + 1)], 'h')
    elif eyes == 'up':
        paint(g, [(ex[0], y0 - 1), (ex[0], y0), (ex[1], y0 - 1), (ex[1], y0)], 'k')
    return g
def cursor():
    out = {}
    out['idle'] = render(cube(), None, CUR_D, CUR_O)
    out['idle2'] = render(cube('blink'), None, CUR_D, CUR_O)
    out['lift'] = render(cube('up', bottom=18), None, CUR_D, CUR_O)  # bobs up, looks up at what it holds
    thr = cube('blink', dx=1, flash=True)  # the facet flashes as it fires
    paint(thr, [(21, 5), (22, 4), (21, 9), (22, 9), (23, 9), (21, 13), (22, 14)], 's')
    out['throw'] = render(thr, None, CUR_D, CUR_O)
    out['hop'] = render(cube('happy'), None, CUR_D, CUR_O)
    return out

# ---------- Gemini: the mark's sparkle, red up, yellow left, green down, blue right and at the heart ----------
def pal4(b, c, d, f): return dict(b=hexc(b), c=hexc(c), d=hexc(d), f=hexc(f))
G_RED = pal4('ffb3ab', 'f47a6e', 'ea4335', 'c5221f')
G_YEL = pal4('ffe9a0', 'fdd663', 'fbbc04', 'e09c00')
G_GRN = pal4('a8e6b8', '5bc27a', '34a853', '1e8e3e')
G_BLU = pal4('b8d3ff', '7baaf7', '4285f4', '2b6ad8')
def gem_at(cx, cy):
    def f(x, y):
        dx, dy = x - cx, y - cy
        if max(abs(dx), abs(dy)) <= 2.5:
            return G_BLU
        if abs(dy) > abs(dx):
            return G_RED if dy < 0 else G_GRN
        return G_YEL if dx < 0 else G_BLU
    return f
GEM_D = {'k': hexc('172036'), 'h': hexc('172036'), 's': hexc('ffffff')}
GEM_O = hexc('172036')
def gemini():
    out = {}
    base = gem_at(11.5, 10)
    out['idle'] = render(face(shift(star(), 0, -2), [(9, 'k'), (14, 'k')], 9), None, GEM_D, GEM_O, base)
    out['idle2'] = render(face(shift(star(0.92, 1.06), 0, -2), [(9, 'k'), (14, 'k')], 9), None, GEM_D, GEM_O, base)
    out['lift'] = render(face(shift(star(rot=math.pi / 8), 0, -3), [(9, 'k'), (14, 'k')], 8), None, GEM_D, GEM_O, gem_at(11.5, 9))
    thr = shift(star(1.22, 0.82), 0, -2)
    paint(thr, [(1, 6), (2, 5), (3, 5), (20, 18), (21, 18), (22, 17)], 's')
    out['throw'] = render(face(thr, [(9, 'k'), (14, 'k')], 9), None, GEM_D, GEM_O, base)
    hop = shift(star(0.9, 1.12), 0, -2)
    paint(hop, [(9, 9), (8, 10), (10, 10), (14, 9), (13, 10), (15, 10)], 'h')
    out['hop'] = render(hop, None, GEM_D, GEM_O, base)
    return out

# ---------- Copilot: the mark's head, goggles pushed up on top, a face with two eyes below ----------
HEAD = [
    '.....##########.....',
    '...##############...',
    '..##RRRRR##RRRRR##..',
    '.##RwLLLLRRwLLLLR##.',
    '.##RLLLLLRRLLLLLR##.',
    '.##RLLLLLRRLLLLLR##.',
    '.###RRRRR##RRRRR###.',
    '####################',
    '###FFFFFFFFFFFFFF###',
    '##FFFFFFFFFFFFFFFF##',
    '##FFFFkFFFFFFkFFFF##',
    '##FFFFkFFFFFFkFFFF##',
    '##FFFFFFFFFFFFFFFF##',
    '.##FFFFFFFFFFFFFF##.',
    '..###FFFFFFFFFF###..',
    '....############....',
    '.....##......##.....',
]
COP = dict(b=hexc('8c959f'), c=hexc('57606a'), d=hexc('424a53'), f=hexc('2d333b'))
COP_D = {'R': hexc('1f2328'), 'L': hexc('b6c8e0'), 'w': hexc('ffffff'), 'F': hexc('e6edf3'), 'k': hexc('1f2328'), 'h': hexc('1f2328')}
COP_O = hexc('101317')
def head(eyes='open', arms=(), dy=0):
    g = place(HEAD, left=2, bottom=22 + dy)
    ks = find(g, 'k')
    if eyes != 'open':
        paint(g, ks, 'F')
        xs = sorted({x for x, y in ks}); y0 = min(y for x, y in ks)
        if eyes == 'blink':
            paint(g, [(x, y0 + 1) for x in xs], 'k')
        else:
            paint(g, [(x, y0) for x in xs] + [(x + d, y0 + 1) for x in xs for d in (-1, 1)], 'h')
    for x, y in arms:
        g[y + dy][x] = '#'
    return g
def copilot():
    out = {}
    out['idle'] = render(head(), COP, COP_D, COP_O)
    out['idle2'] = render(head('blink'), COP, COP_D, COP_O)
    up = [(x, y) for x in (0, 1) for y in range(4, 11)] + [(x, y) for x in (22, 23) for y in range(4, 11)]
    out['lift'] = render(head(arms=up, dy=-1), COP, COP_D, COP_O)
    fwd = [(x, y) for x in (22, 23) for y in (15, 16)] + [(x, y) for x in (0, 1) for y in (17, 18)]
    out['throw'] = render(head(arms=fwd), COP, COP_D, COP_O)
    hop = [(1, 9), (0, 8), (1, 8), (22, 9), (23, 8), (22, 8)]
    out['hop'] = render(head('happy', arms=hop), COP, COP_D, COP_O)
    return out

# ---------- OpenCode: the mark's thick frame, its opening a pale panel with the face ----------
FRAME = [
    '#############',
    '#############',
    '#############',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '###PPPPPPP###',
    '#############',
    '#############',
    '#############',
    '.###.....###.',
]
OC = dict(b=hexc('8a8a96'), c=hexc('5d5d68'), d=hexc('3e3e47'), f=hexc('2a2a31'))
OC_D = {'P': hexc('d9dbe2'), 'Q': hexc('f4f5fa'), 'W': hexc('ffffff'), 'k': hexc('1b1b20'), 'h': hexc('1b1b20'), 'm': hexc('1b1b20')}
OC_O = hexc('0f0f13')
def frame(eyes='open', panel='P', dy=0, lean=0):
    g = place(FRAME, bottom=22 + dy)
    ps = find(g, 'P')
    paint(g, ps, panel)
    x0 = min(x for x, y in ps); y0 = min(y for x, y in ps)
    e = [(x0 + 1, y0 + 3), (x0 + 5, y0 + 3)]
    if eyes == 'open':
        paint(g, [(x, y) for x, y in e] + [(x, y + 1) for x, y in e], 'k')
    elif eyes == 'blink':
        paint(g, [(x, y + 1) for x, y in e], 'k')
    elif eyes == 'happy':
        paint(g, [(x, y) for x, y in e] + [(x + d, y + 1) for x, y in e for d in (-1, 1)], 'h')
    if eyes in ('open', 'happy'):
        paint(g, [(x0 + 3, y0 + 6)], 'm')
    if lean:
        top = min(y for x, y in find(g, '#'))
        for y in range(top, top + 8):
            g[y] = (['.'] * lean + g[y])[:N]
    return g
def opencode():
    out = {}
    out['idle'] = render(frame(), OC, OC_D, OC_O)
    out['idle2'] = render(frame('blink'), OC, OC_D, OC_O)
    out['lift'] = render(frame(panel='Q', dy=-1), OC, OC_D, OC_O)   # the opening lights up
    out['throw'] = render(frame('blink', panel='W', lean=1), OC, OC_D, OC_O)  # and flashes as it spits
    out['hop'] = render(frame('happy'), OC, OC_D, OC_O)
    return out

if __name__ == '__main__':
    dest = sys.argv[1]
    for who, fn in (('gemini', gemini), ('cursor', cursor), ('copilot', copilot), ('opencode', opencode)):
        os.makedirs(f'{dest}/{who}', exist_ok=True)
        for pose, im in fn().items():
            im.save(f'{dest}/{who}/{pose}.png')
