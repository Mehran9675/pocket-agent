# Draws assets/appicon.png: an 80s pixel-art face in sunglasses looking
# cool to the left, on eggshell. A 128x128 grid scaled to 1024px; the
# face keeps to the middle ~55% so launcher masks (One UI's squircle,
# Android adaptive icons) never clip it. Shapes are defined relative to
# the face radius, so the grid size can change.
#   python3 scripts/appicon.py assets/appicon.png
import math
import sys

from PIL import Image

N = 128
S = 1024 // N
BG = (240, 234, 214)          # eggshell
OUT = (52, 30, 22)            # outline
FACE = (255, 204, 51)
HI = (255, 230, 120)          # highlight
HI2 = (255, 247, 200)         # specular
SH = (240, 170, 40)           # shade
SH2 = (214, 132, 30)          # deep shade
FRAME = (20, 18, 32)          # sunglasses
LENS = [(30, 22, 70), (52, 28, 98), (88, 34, 126), (140, 44, 146), (200, 60, 150), (245, 100, 130), (255, 150, 110)]
GLINT = (240, 250, 255)
MOUTH = (110, 46, 24)

g = [[None] * N for _ in range(N)]
R = N * 0.275                 # face radius: diameter 55% of the icon
cx, cy = N / 2 - 0.5, N / 2 + 0.5
look = -R * 0.16              # glasses and mouth sit left: he looks left


def put(x, y, c):
    if 0 <= x < N and 0 <= y < N:
        g[y][x] = c


def on_face(x, y, inset=0.0):
    return math.hypot(x - cx, y - cy) <= R - inset


# Face: a shaded disc lit from the top left, with a two-cell outline.
for y in range(N):
    for x in range(N):
        d = math.hypot(x - cx, y - cy)
        if d > R:
            continue
        if d > R - 2.2:
            g[y][x] = OUT
            continue
        lx, ly = (x - cx) / R, (y - cy) / R
        lit = -0.6 * lx - 0.8 * ly  # 1 at the top left, -1 at the bottom right
        rim = (R - d) / R           # 0 at the edge, 1 at the center
        c = FACE
        if lit > 0.45 and rim < 0.32:
            c = HI
        if lit < -0.3 and rim < 0.28:
            c = SH
        if lit < -0.5 and rim < 0.13:
            c = SH2
        g[y][x] = c
# Specular highlight: a small curved streak at the top left.
for t in range(40):
    a = math.radians(205 + t * 1.3)
    for w in (0, 1):
        r = R - 6 - w
        put(round(cx + r * math.cos(a)), round(cy + r * math.sin(a)), HI2)

# Sunglasses: a bar across the face and two equal lenses under it.
bar_y = round(cy - R * 0.26)
bar_h = 3
for y in range(bar_y, bar_y + bar_h):
    for x in range(N):
        if on_face(x, y, 1.0):
            put(x, y, FRAME)

lens_w = round(R * 0.56)
lens_h = round(R * 0.42)
gap = round(R * 0.16)
mid = round(cx + look)
lenses = [(mid - gap // 2 - lens_w, mid - gap // 2 - 1), (mid + (gap + 1) // 2, mid + (gap + 1) // 2 + lens_w - 1)]


def lens(x0, x1):
    """A lens hanging from the bar: flat top, rounded bottom corners."""
    top = bar_y + bar_h
    w = x1 - x0 + 1
    r = lens_h * 0.55  # bottom corner radius
    for i in range(lens_h):
        y = top + i
        # Inset of the rounded bottom at this row.
        dy = i - (lens_h - r)
        inset = 0 if dy <= 0 else round(r - math.sqrt(max(0.0, r * r - dy * dy)))
        a, b = x0 + inset, x1 - inset
        for x in range(a, b + 1):
            put(x, y, FRAME)
        # Reflection bands inside a one-cell frame.
        if 1 <= i < lens_h - 1:
            band = LENS[min(len(LENS) - 1, (i - 1) * len(LENS) // (lens_h - 2))]
            for x in range(a + 1, b):
                put(x, y, band)
    # Glint: two short diagonal streaks at the top left of the lens.
    for k in range(5):
        put(x0 + 3 + k, top + 6 - k, GLINT)
        put(x0 + 4 + k, top + 6 - k, GLINT)
    for k in range(2):
        put(x0 + 10 + k, top + 3 - k, GLINT)


for x0, x1 in lenses:
    lens(x0, x1)
# Bridge between the lenses, under the bar.
for x in range(lenses[0][1], lenses[1][0] + 1):
    for y in range(bar_y + bar_h, bar_y + bar_h + 2):
        put(x, y, FRAME)

# A small smirk centered under the glasses, raised at its left end.
my = round(cy + R * 0.42)
half = round(R * 0.2)
for x in range(mid - half, mid + half + 1):
    t = (x - (mid - half)) / (2 * half)  # 0 at the left end
    lift = round(max(0.0, 0.35 - t) * 9)  # the left end curls up
    for w in (0, 1):
        put(x, my - lift + w, MOUTH)

img = Image.new("RGB", (N * S, N * S), BG)
px = img.load()
for y in range(N):
    for x in range(N):
        c = g[y][x]
        if c is None:
            continue
        for dy in range(S):
            for dx in range(S):
                px[x * S + dx, y * S + dy] = c
img.save(sys.argv[1])
