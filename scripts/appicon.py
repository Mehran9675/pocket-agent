# Draws assets/appicon.png: an 80s pixel-art face in sunglasses looking
# cool to the left, on eggshell. A 64x64 grid scaled to 1024px; the face
# keeps to the middle ~55% so launcher masks (One UI's squircle, Android
# adaptive icons) never clip it.
#   python3 scripts/appicon.py assets/appicon.png
import math
import sys

from PIL import Image

N, S = 64, 16
BG = (240, 234, 214)          # eggshell
OUT = (52, 30, 22)            # outline
FACE = (255, 204, 51)
HI = (255, 236, 140)          # highlight
HI2 = (255, 250, 210)         # specular
SH = (236, 160, 36)           # shade
SH2 = (205, 120, 30)          # deep shade
FRAME = (20, 18, 32)          # sunglasses
LENS = [(34, 24, 74), (64, 32, 110), (120, 40, 140), (214, 64, 150), (255, 120, 120)]  # synthwave sky
GLINT = (240, 250, 255)
MOUTH = (110, 46, 24)

g = [[None] * N for _ in range(N)]
cx, cy, R = 31.5, 33.5, 17.5


def put(x, y, c):
    if 0 <= x < N and 0 <= y < N:
        g[y][x] = c


# Face: shaded disc with an outline.
for y in range(N):
    for x in range(N):
        d = math.hypot(x - cx, y - cy)
        if d > R:
            continue
        if d > R - 1.2:
            g[y][x] = OUT
            continue
        # Light from the top left.
        lx, ly = (x - cx) / R, (y - cy) / R
        lit = -0.6 * lx - 0.8 * ly
        c = FACE
        if lit > 0.55 and d > R - 6:
            c = HI
        if lit < -0.35 and d > R - 5:
            c = SH
        if lit < -0.55 and d > R - 2.6:
            c = SH2
        g[y][x] = c
# Specular glint, top left.
for x, y in [(21, 21), (22, 20), (23, 20), (21, 22), (24, 19)]:
    put(x, y, HI2)

# Sunglasses, turned left: the near lens (left) is wider than the far one.
shift = -3
bar_y = 29
for x in range(int(cx - R) + 1, int(cx + R) - 0):
    for y in (bar_y, bar_y + 1):
        if math.hypot(x - cx, y - cy) < R - 0.6:
            put(x, y, FRAME)


def lens(x0, x1, rows):
    """A lens from x0 to x1 (inclusive) under the bar, narrowing at the bottom."""
    for i in range(rows):
        inset = 0 if i < rows - 3 else (i - (rows - 4))
        for x in range(x0 + inset, x1 - inset + 1):
            put(x, bar_y + 2 + i, FRAME)
        # Reflection bands inside the frame.
        if 0 < i < rows - 1:
            band = LENS[min(len(LENS) - 1, i * len(LENS) // (rows - 1))]
            for x in range(x0 + inset + 1, x1 - inset):
                put(x, bar_y + 2 + i, band)


near = (int(cx) + shift - 12, int(cx) + shift - 1)
far = (int(cx) + shift + 2, int(cx) + shift + 11)
lens(near[0], near[1], 8)
lens(far[0], far[1], 7)
# Bridge.
for x in range(near[1], far[0] + 1):
    put(x, bar_y + 2, FRAME)
# Glints on the lenses: a short diagonal streak each.
for ox in (near[0] + 2, far[0] + 2):
    for k in range(3):
        put(ox + k, bar_y + 3 + 2 - k, GLINT)

# A small smirk, off to the left.
smirk = [(23, 43), (24, 43), (25, 43), (26, 43), (27, 43), (28, 42), (29, 41)]
for x, y in smirk:
    put(x + shift + 2, y, MOUTH)

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
