import os
from PIL import Image, ImageDraw, ImageFilter, ImageFont

S = 4
N = 74 * S
OUT = os.path.join("dist", "Tractor", "icon.png")

img = Image.new("RGBA", (N, N), (0,0,0,0))
grad = Image.new("RGBA", (N, N))
d = ImageDraw.Draw(grad)
for y in range(N):
    t = y/(N-1)
    col = tuple(int(a*(1-t)+b*t) for a,b in zip((26,92,70),(7,34,26)))+(255,)
    d.line([(0,y),(N,y)], fill=col)
mask = Image.new("L",(N,N),0)
ImageDraw.Draw(mask).rounded_rectangle([0,0,N-1,N-1], radius=48, fill=255)
img.paste(grad,(0,0),mask)
d = ImageDraw.Draw(img)
d.rounded_rectangle([4,4,N-5,N-5], radius=44, outline=(233,196,106), width=8)

cw, ch = 168, 224
x0, y0 = (N-cw)//2, (N-ch)//2

sh = Image.new("RGBA",(N,N),(0,0,0,0))
ImageDraw.Draw(sh).rounded_rectangle([x0+6,y0+12,x0+cw+6,y0+ch+12], radius=20, fill=(0,0,0,120))
sh = sh.filter(ImageFilter.GaussianBlur(12))
img.alpha_composite(sh)

cd = ImageDraw.Draw(img)
cd.rounded_rectangle([x0,y0,x0+cw,y0+ch], radius=20, fill=(251,247,235,255), outline=(216,206,180,255), width=4)
font = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 112)
t="拖"
bb = cd.textbbox((0,0),t,font=font)
cd.text((x0+(cw-(bb[2]-bb[0]))/2-bb[0], y0+(ch-(bb[3]-bb[1]))/2-bb[1]), t, font=font, fill=(200,59,46))
sf = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 46)
cd.text((x0+cw-58, y0+ch-64), "♠", font=sf, fill=(28,28,28))

img = img.resize((74,74), Image.LANCZOS)
img.save(OUT)
print("wrote", OUT)
