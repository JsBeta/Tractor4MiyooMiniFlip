"""ASCII preview of a PNG screenshot, for quick layout checks."""
import sys

from PIL import Image

CHARS = " .:-=+*#%@"


def main(path, width=110):
    img = Image.open(path).convert("L")
    w, h = img.size
    nh = max(1, int(h * width / w / 2.1))
    small = img.resize((width, nh), Image.LANCZOS)
    print("%s  %dx%d" % (path, w, h))
    px = small.load()
    for y in range(nh):
        line = []
        for x in range(width):
            v = px[x, y]
            line.append(CHARS[min(len(CHARS) - 1, v * len(CHARS) // 256)])
        print("".join(line))


if __name__ == "__main__":
    main(sys.argv[1], int(sys.argv[2]) if len(sys.argv) > 2 else 110)
