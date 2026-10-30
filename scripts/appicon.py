# Draws assets/appicon.png: an AI sparkle rising out of a stitched
# pocket on an indigo-violet gradient - private AI in your pocket.
# Drawn 4x larger and scaled down for smooth edges. The motif keeps to
# the middle ~60% so launcher masks (One UI's squircle, Android
# adaptive icons) never clip it.
#   python3 scripts/appicon.py assets/appicon.png
import math
import sys

from PIL import Image, ImageChops, ImageDraw, ImageFilter

OUT = 1024
SS = 4
N = OUT * SS
C = N / 2

BG_TOP = (99, 102, 241)       # indigo
BG_BOTTOM = (139, 92, 246)    # violet
POCKET = (250, 249, 255)
POCKET_SHADE = (226, 224, 252)
STITCH = (124, 112, 236)
SPARK = (255, 209, 102)       # warm gold
SPARK_CORE = (255, 244, 214)
SHADOW = (40, 24, 110)


def u(v):
    return v * N


def gradient(a, b):
    """A top-left to bottom-right blend of a and b."""
    v = Image.linear_gradient("L").resize((N, N))
    g = ImageChops.add(v, v.transpose(Image.Transpose.ROTATE_90).transpose(Image.Transpose.FLIP_LEFT_RIGHT), scale=2)
    return Image.composite(Image.new("RGB", (N, N), b), Image.new("RGB", (N, N), a), g)


def sparkle(cx, cy, r, p=2.4, steps=720):
    """A four-point star with concave sides."""
    pts = []
    for i in range(steps):
        t = 2 * math.pi * i / steps
        c, s = math.cos(t), math.sin(t)
        pts.append((cx + r * math.copysign(abs(c) ** p, c), cy + r * math.copysign(abs(s) ** p, s)))
    return pts


def pocket(cx, top, w, h, inset=0.0, steps=200):
    """A jeans pocket: straight top, straight sides, curving to a soft
    point at the bottom."""
    w2 = w / 2 - inset
    t = top + inset
    side = top + h * 0.5
    bottom = top + h - inset * 1.25
    pts = [(cx - w2, t), (cx + w2, t), (cx + w2, side)]
    # Right side to the bottom point, then mirrored.
    for i in range(1, steps + 1):
        k = i / steps
        x = cx + w2 * (1 - k) ** 1.15
        y = side + (bottom - side) * math.sin(k * math.pi / 2) ** 0.9
        pts.append((x, y))
    left = [(2 * cx - x, y) for x, y in reversed(pts[3:-1])]
    return pts + left + [(cx - w2, side)]


def dashed(draw, pts, dash, gap, width, fill):
    """Strokes the open path pts with round-capped dashes."""
    dashes, cur, on, left = [], [pts[0]], True, dash
    for (x0, y0), (x1, y1) in zip(pts, pts[1:]):
        seg = math.hypot(x1 - x0, y1 - y0)
        pos = 0.0
        while pos < seg:
            step = min(left, seg - pos)
            pos += step
            left -= step
            p = (x0 + (x1 - x0) * pos / seg, y0 + (y1 - y0) * pos / seg)
            if on:
                cur.append(p)
            if left <= 0:
                if on:
                    dashes.append(cur)
                on = not on
                left = dash if on else gap
                cur = [p]
    if on and len(cur) > 1:
        dashes.append(cur)
    r = width / 2
    for ds in dashes:
        draw.line(ds, fill=fill, width=width, joint="curve")
        for x, y in (ds[0], ds[-1]):
            draw.ellipse([x - r, y - r, x + r, y + r], fill=fill)


def main(path):
    img = gradient(BG_TOP, BG_BOTTOM)

    # Geometry, as fractions of the icon.
    pw, ph = u(0.44), u(0.33)
    ptop = u(0.48)
    sr = u(0.19)
    scx, scy = C, ptop - u(0.09)

    # The sparkle sits behind the pocket, rising out of it, with a glow.
    glow = Image.new("L", (N, N), 0)
    ImageDraw.Draw(glow).polygon(sparkle(scx, scy, sr * 1.08), fill=150)
    glow = glow.filter(ImageFilter.GaussianBlur(u(0.03)))
    img = Image.composite(Image.new("RGB", (N, N), SPARK), img, glow)
    d = ImageDraw.Draw(img)
    d.polygon(sparkle(scx, scy, sr), fill=SPARK)
    d.polygon(sparkle(scx, scy, sr * 0.4, p=2.8), fill=SPARK_CORE)
    # A small companion sparkle.
    d.polygon(sparkle(C + u(0.17), u(0.24), u(0.055)), fill=SPARK_CORE)

    # The pocket's soft shadow, then the pocket.
    outline = pocket(C, ptop, pw, ph)
    shadow = Image.new("L", (N, N), 0)
    ImageDraw.Draw(shadow).polygon([(x, y + u(0.018)) for x, y in outline], fill=120)
    shadow = shadow.filter(ImageFilter.GaussianBlur(u(0.025)))
    img = Image.composite(Image.new("RGB", (N, N), SHADOW), img, shadow)

    mask = Image.new("L", (N, N), 0)
    ImageDraw.Draw(mask).polygon(outline, fill=255)
    # Rounded top corners.
    r = u(0.03)
    md = ImageDraw.Draw(mask)
    for sx in (-1, 1):
        x = C + sx * pw / 2
        md.rectangle([min(x, x - sx * r), ptop, max(x, x - sx * r), ptop + r], fill=0)
        md.ellipse([x - sx * r - r, ptop, x - sx * r + r, ptop + 2 * r], fill=255)
    body = gradient(POCKET, POCKET_SHADE)
    img = Image.composite(body, img, mask)

    # Stitching: a hem across the top and a seam inset along the sides
    # and bottom.
    d = ImageDraw.Draw(img)
    sw = int(u(0.009))
    hem = ptop + u(0.07)
    dashed(d, [(C - pw / 2 + u(0.045), hem), (C + pw / 2 - u(0.045), hem)], u(0.03), u(0.02), sw, STITCH)
    seam = pocket(C, ptop, pw, ph, inset=u(0.04))
    open_seam = seam[1:] + [seam[0]]
    open_seam = [(x, max(y, hem + u(0.04))) for x, y in open_seam]
    dashed(d, open_seam, u(0.03), u(0.02), sw, STITCH)

    img.resize((OUT, OUT), Image.LANCZOS).save(path)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "assets/appicon.png")
