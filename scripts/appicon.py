# Draws assets/appicon.png: 80s pixel-art emoji with sunglasses on eggshell,
# a 32x32 grid scaled to 1024px. Usage: python3 scripts/appicon.py assets/appicon.png
import sys, math
from PIL import Image
N, S = 32, 32
BG = (240, 234, 214)        # eggshell
OUT = (58, 34, 20); FACE = (255, 204, 51); HI = (255, 238, 150); SH = (228, 150, 32)
K = (22, 20, 34); CY = (80, 220, 255); PK = (255, 64, 170); M = (122, 42, 20); T = (255, 255, 255)
g = [[None] * N for _ in range(N)]
c, R = 15.5, 11.2
for y in range(N):
    for x in range(N):
        d = math.hypot(x - c, y - c)
        if d <= R - 1:
            g[y][x] = FACE
            # rim lighting: highlight upper-left, shade lower-right
            if d > R - 2.6:
                a = math.atan2(y - c, x - c)
                if -2.9 < a < -1.6: g[y][x] = HI
                elif 0.25 < a < 1.3 and d > R - 2.0: g[y][x] = SH
        elif d <= R:
            g[y][x] = OUT
def mirror(y, half, pal):
    # half covers columns 0..15; mirrored to 16..31
    for x, ch in enumerate(half):
        if ch == ' ': continue
        g[y][x] = g[y][N - 1 - x] = pal[ch]
P = {'k': K, 'o': OUT, 'y': FACE, 'm': M, 't': T}
# sunglasses: bar across, two trapezoid lenses, bridge
glasses = [
    (12, "    kkkkkkkkkkkk"),
    (13, "    kkkkkkkkkkkk"),
    (14, "     kkkkkkkk  k"),
    (15, "     kkkkkkkk   "),
    (16, "      kkkkkk    "),
    (17, "       kkkk     "),
]
for y, h in glasses: mirror(y, h, P)
# lens glints (same spot on each lens)
for ox in (6, 18):
    g[14][ox + 1] = CY; g[14][ox + 2] = PK; g[15][ox + 1] = PK
# grin
mouth = [
    (20, "         m      "),
    (21, "         mmmmmmm"),
    (22, "          mttttt"),
    (23, "           mmmmm"),
]
for y, h in mouth: mirror(y, h, P)
img = Image.new('RGB', (N * S, N * S), BG)
px = img.load()
for y in range(N):
    for x in range(N):
        col = g[y][x]
        if col is None: continue
        for dy in range(S):
            for dx in range(S):
                px[x * S + dx, y * S + dy] = col
img.save(sys.argv[1])
