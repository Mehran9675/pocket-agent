# Draws the app icon: a little robot agent peeking out of a denim
# pocket, gripping its edge - Pocket Agent, literally. Writes
#   assets/appicon.png              the full icon (iOS, older Android)
#   assets/android-foreground.png   adaptive icon layers (Android 8+):
#   assets/android-background.png     artwork in the middle 66 of 108dp
#   assets/android-monochrome.png   the silhouette for themed icons
# Drawn 4x larger and scaled down for smooth edges.
#   python3 scripts/appicon.py [assets]
import math
import os
import sys

from PIL import Image, ImageChops, ImageDraw, ImageFilter

OUT = 1024
SS = 4
N = OUT * SS

BG_A = (255, 138, 92)         # coral
BG_B = (255, 190, 92)         # amber
DENIM = (58, 94, 168)
DENIM_DARK = (40, 66, 128)
STITCH = (255, 196, 110)
RIVET = (214, 150, 84)
ROBOT = (248, 248, 252)
ROBOT_SHADE = (214, 218, 232)
VISOR = (24, 30, 58)
EYE = (110, 240, 255)
SHADOW = (20, 24, 60)


def gradient(a, b):
    """A top-left to bottom-right blend of a and b."""
    v = Image.linear_gradient("L").resize((N, N))
    g = ImageChops.add(v, v.transpose(Image.Transpose.ROTATE_90).transpose(Image.Transpose.FLIP_LEFT_RIGHT), scale=2)
    return Image.composite(Image.new("RGB", (N, N), b), Image.new("RGB", (N, N), a), g)


def rrect(d, box, r, fill):
    d.rounded_rectangle(box, radius=r, fill=fill)


def pocket_shape(x0, top, w, h, steps=160):
    """A jeans pocket: straight top and sides, curving to a soft point."""
    cx, side, bottom = x0 + w / 2, top + h * 0.45, top + h
    pts = [(x0, top), (x0 + w, top), (x0 + w, side)]
    right = []
    for i in range(1, steps + 1):
        k = i / steps
        right.append((cx + (w / 2) * (1 - k) ** 1.2, side + (bottom - side) * math.sin(k * math.pi / 2) ** 0.85))
    left = [(2 * cx - x, y) for x, y in reversed(right[:-1])]
    return pts + right + left + [(x0, side)]


def dashed(d, pts, dash, gap, width, fill):
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
        d.line(ds, fill=fill, width=width, joint="curve")
        for x, y in (ds[0], ds[-1]):
            d.ellipse([x - r, y - r, x + r, y + r], fill=fill)


def inset(pts, cx, cy, by):
    """Shrinks a convex outline toward (cx, cy) by about by."""
    out = []
    for x, y in pts:
        dx, dy = x - cx, y - cy
        l = math.hypot(dx, dy) or 1
        out.append((x - dx / l * by, y - dy / l * by))
    return out


def layer():
    return Image.new("RGBA", (N, N), (0, 0, 0, 0))


def soft(mask, blur):
    return mask.filter(ImageFilter.GaussianBlur(blur))


def art(scale=1.0, mono=False):
    """The robot and pocket on transparency. scale shrinks it about the
    centre; mono draws a white silhouette with the face cut out."""
    img = layer()
    c = N / 2

    def u(v):
        return v * N * scale

    def X(f):  # a horizontal position, as a fraction of the icon
        return c + (f - 0.5) * N * scale

    def Y(f):
        return c + (f - 0.5) * N * scale

    white = (255, 255, 255, 255)
    d = ImageDraw.Draw(img)

    # Antenna.
    ax, atop = X(0.5), Y(0.175)
    d.line([(ax, atop), (ax, Y(0.27))], fill=white if mono else ROBOT_SHADE + (255,), width=int(u(0.016)))
    if not mono:
        glow = Image.new("L", (N, N), 0)
        ImageDraw.Draw(glow).ellipse([ax - u(0.05), atop - u(0.05), ax + u(0.05), atop + u(0.05)], fill=170)
        img = Image.composite(Image.new("RGBA", (N, N), EYE + (255,)), img, soft(glow, u(0.02)))
        d = ImageDraw.Draw(img)
    d.ellipse([ax - u(0.03), atop - u(0.03), ax + u(0.03), atop + u(0.03)], fill=white if mono else EYE + (255,))

    # Head: its bottom is hidden by the pocket.
    hx0, hx1, hy0, hy1 = X(0.30), X(0.70), Y(0.255), Y(0.62)
    rrect(d, [hx0, hy0, hx1, hy1], u(0.09), white if mono else ROBOT + (255,))
    if not mono:
        # A little shading down the right side.
        shade = Image.new("L", (N, N), 0)
        sd = ImageDraw.Draw(shade)
        sd.rounded_rectangle([hx0, hy0, hx1, hy1], radius=u(0.09), fill=255)
        cut = Image.new("L", (N, N), 0)
        ImageDraw.Draw(cut).rounded_rectangle([hx0 - u(0.03), hy0 - u(0.03), hx1 - u(0.035), hy1], radius=u(0.1), fill=255)
        shade = ImageChops.subtract(shade, cut)
        img = Image.composite(Image.new("RGBA", (N, N), ROBOT_SHADE + (255,)), img, soft(shade, u(0.006)))
        d = ImageDraw.Draw(img)

    # Visor and eyes, looking up at you.
    vx0, vx1, vy0, vy1 = X(0.345), X(0.655), Y(0.315), Y(0.465)
    if mono:
        cutout = Image.new("L", (N, N), 0)
        cd = ImageDraw.Draw(cutout)
        for ex in (0.43, 0.57):
            cd.rounded_rectangle([X(ex - 0.03), Y(0.35), X(ex + 0.03), Y(0.43)], radius=u(0.03), fill=255)
        img.putalpha(ImageChops.subtract(img.getchannel("A"), cutout))
        d = ImageDraw.Draw(img)
    else:
        rrect(d, [vx0, vy0, vx1, vy1], u(0.06), VISOR + (255,))
        glow = Image.new("L", (N, N), 0)
        gd = ImageDraw.Draw(glow)
        for ex in (0.43, 0.57):
            gd.rounded_rectangle([X(ex - 0.04), Y(0.34), X(ex + 0.04), Y(0.44)], radius=u(0.04), fill=200)
        glow = ImageChops.multiply(soft(glow, u(0.018)), Image.new("L", (N, N), 255))
        vmask = Image.new("L", (N, N), 0)
        ImageDraw.Draw(vmask).rounded_rectangle([vx0, vy0, vx1, vy1], radius=u(0.06), fill=255)
        img = Image.composite(Image.new("RGBA", (N, N), EYE + (255,)), img, ImageChops.multiply(glow, vmask))
        d = ImageDraw.Draw(img)
        for ex in (0.43, 0.57):
            rrect(d, [X(ex - 0.026), Y(0.352), X(ex + 0.026), Y(0.428)], u(0.026), EYE + (255,))
            d.ellipse([X(ex - 0.012), Y(0.36), X(ex + 0.002), Y(0.374)], fill=(235, 255, 255, 255))

    # The pocket, in front, with its shadow on the head.
    px0, pw, ptop, ph = X(0.22), u(0.56), Y(0.52), u(0.31)
    outline = pocket_shape(px0, ptop, pw, ph)
    if not mono:
        sh = Image.new("L", (N, N), 0)
        ImageDraw.Draw(sh).polygon([(x, y - u(0.012)) for x, y in outline], fill=150)
        sh = ImageChops.multiply(soft(sh, u(0.02)), img.getchannel("A"))
        img = Image.composite(Image.new("RGBA", (N, N), SHADOW + (255,)), img, sh)
    pm = Image.new("L", (N, N), 0)
    ImageDraw.Draw(pm).polygon(outline, fill=255)
    if mono:
        img = Image.composite(Image.new("RGBA", (N, N), white), img, pm)
        # A gap between head and pocket keeps them apart in the silhouette.
        gap = Image.new("L", (N, N), 0)
        ImageDraw.Draw(gap).line([(px0, ptop - u(0.006)), (px0 + pw, ptop - u(0.006))], fill=255, width=int(u(0.016)))
        img.putalpha(ImageChops.subtract(img.getchannel("A"), gap))
    else:
        body = gradient(DENIM, DENIM_DARK).convert("RGBA")
        img = Image.composite(body, img, pm)
        d = ImageDraw.Draw(img)
        # Hem band, stitching and rivets.
        hem = ptop + u(0.06)
        d.line([(px0, hem), (px0 + pw, hem)], fill=DENIM_DARK + (255,), width=int(u(0.006)))
        sw = int(u(0.009))
        dashed(d, [(px0 + u(0.035), ptop + u(0.03)), (px0 + pw - u(0.035), ptop + u(0.03))], u(0.026), u(0.018), sw, STITCH + (255,))
        cx = px0 + pw / 2
        seam = inset(outline[2:-1], cx, ptop + ph * 0.35, u(0.04))
        seam = [(x, y) for x, y in seam if y > hem + u(0.03)]
        dashed(d, seam, u(0.026), u(0.018), sw, STITCH + (255,))
        for rx in (px0 + u(0.035), px0 + pw - u(0.035)):
            d.ellipse([rx - u(0.017), hem + u(0.02) - u(0.017), rx + u(0.017), hem + u(0.02) + u(0.017)], fill=RIVET + (255,))

    # Hands gripping the pocket's edge.
    for hx in (0.33, 0.67):
        x0, x1 = X(hx - 0.06), X(hx + 0.06)
        y0, y1 = Y(0.495), Y(0.565)
        rrect(d, [x0, y0, x1, y1], u(0.035), white if mono else ROBOT + (255,))
        if not mono:
            for f in (-1, 0, 1):
                fx = X(hx + f * 0.028)
                d.line([(fx, Y(0.548)), (fx, Y(0.565))], fill=ROBOT_SHADE + (255,), width=int(u(0.006)))
    return img


def down(img):
    return img.resize((OUT, OUT), Image.LANCZOS)


def main(dirpath):
    bg = gradient(BG_A, BG_B).convert("RGBA")
    full = Image.alpha_composite(bg, art())
    down(full).convert("RGB").save(os.path.join(dirpath, "appicon.png"))
    # Adaptive icons keep the middle 66 of 108dp safe: 0.61 of the
    # canvas. The artwork spans 0.69 of the icon, so it shrinks.
    down(art(scale=0.84)).save(os.path.join(dirpath, "android-foreground.png"))
    down(bg).convert("RGB").save(os.path.join(dirpath, "android-background.png"))
    down(art(scale=0.84, mono=True)).save(os.path.join(dirpath, "android-monochrome.png"))


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else "assets")
