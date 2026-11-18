"""Build the embedded bitmap font atlases for the Tractor port.

Scans the Go sources for string literals, collects every rune used by the UI and
rasterises three sizes with the Windows CJK font. Run from the project root:

    python tools/genfont.py
"""
import json
import os
import re
import sys

from PIL import Image, ImageDraw, ImageFont

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "internal", "gfx")
FONT_PATH = r"C:\Windows\Fonts\msyh.ttc"

SIZES = [("xs", 16), ("sm", 20), ("md", 26), ("lg", 32), ("xl", 52)]

BASE = (
    "".join(chr(c) for c in range(32, 127))
    + "♠♥♣♦"
    + "▲▼◀▶"
    + "×÷·—…、，。：；！？（）【】《》“”‘’"
    + "①②③④⑤⑥⑦⑧⑨⑩"
    + "─│┌┐└┘"
)

STRING_RE = re.compile(r'"((?:[^"\\\n]|\\.)*)"|`([^`]*)`')


def collect_chars():
    chars = set(BASE)
    for sub in ("internal", "cmd"):
        for dirpath, _dirs, files in os.walk(os.path.join(ROOT, sub)):
            for name in files:
                if not name.endswith(".go"):
                    continue
                path = os.path.join(dirpath, name)
                text = open(path, encoding="utf-8").read()
                for m in STRING_RE.finditer(text):
                    s = m.group(1) or m.group(2) or ""
                    for ch in s:
                        if ch >= " ":
                            chars.add(ch)
    # Drop runs that are only whitespace.
    return sorted(chars)


def render(font, ch, line, pad=2):
    try:
        bbox = font.getbbox(ch)
    except Exception:
        bbox = None
    adv = max(1, int(round(font.getlength(ch))))
    if not bbox or bbox[2] <= bbox[0] or bbox[3] <= bbox[1]:
        return Image.new("L", (1, 1), 0), (0, 0, 0, 0), adv
    x0, y0, x1, y1 = bbox
    w, h = x1 - x0, y1 - y0
    img = Image.new("L", (w + pad * 2, h + pad * 2), 0)
    d = ImageDraw.Draw(img)
    d.text((pad - x0, pad - y0), ch, font=font, fill=255)
    return img, (x0, y0, w, h), adv


def build(name, size, chars):
    font = ImageFont.truetype(FONT_PATH, size)
    ascent, descent = font.getmetrics()
    line = ascent + descent
    glyph_images = []
    for ch in chars:
        img, (x0, y0, w, h), adv = render(font, ch, line)
        if w <= 0 or h <= 0:
            glyph_images.append((ch, None, adv, 0, 0))
            continue
        glyph_images.append((ch, img, adv, x0, y0))

    # Shelf packing.
    width = 1024
    pad = 1
    x = y = pad
    shelf = 0
    placed = []
    for ch, img, adv, x0, y0 in glyph_images:
        if img is None:
            continue
        w, h = img.size
        if x + w + pad > width:
            x = pad
            y += shelf + pad
            shelf = 0
        placed.append((ch, img, adv, x0, y0, x, y))
        x += w + pad
        shelf = max(shelf, h)
    height = 1
    while height < y + shelf + pad:
        height *= 2

    atlas = Image.new("L", (width, height), 0)
    glyphs = {}
    for ch, img, adv, x0, y0, gx, gy in placed:
        atlas.paste(img, (gx, gy))
        glyphs[ch] = {"x": gx, "y": gy, "w": img.size[0], "h": img.size[1],
                      "ox": x0, "oy": y0, "adv": adv}

    out_png = os.path.join(OUT, "font_%s.png" % name)
    out_json = os.path.join(OUT, "font_%s.json" % name)
    atlas.save(out_png, optimize=True)
    meta = {"size": size, "line": line, "ascent": ascent, "glyphs": glyphs}
    with open(out_json, "w", encoding="utf-8") as f:
        json.dump(meta, f, ensure_ascii=False, separators=(",", ":"))
    print("%s: %d glyphs, atlas %dx%d, png %d bytes" % (
        name, len(glyphs), width, height, os.path.getsize(out_png)))


def main():
    chars = collect_chars()
    print("charset: %d runes" % len(chars))
    missing = [c for c in chars if ord(c) > 0x2E80 and c not in BASE]
    print("cjk: %d" % len(missing))
    for name, size in SIZES:
        build(name, size, chars)


if __name__ == "__main__":
    main()
