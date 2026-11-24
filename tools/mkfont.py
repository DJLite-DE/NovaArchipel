import struct, sys
from PIL import Image, ImageDraw, ImageFont
G='/usr/share/fonts/truetype/google-fonts/'
O=G+'ofl/'
fonts=[(G+'Poppins-Medium.ttf',12),(G+'Poppins-Medium.ttf',14),(G+'Poppins-SemiBold.ttf' if __import__('os').path.exists(G+'Poppins-SemiBold.ttf') else G+'Poppins-Bold.ttf',16),(G+'Poppins-SemiBold.ttf' if __import__('os').path.exists(G+'Poppins-SemiBold.ttf') else G+'Poppins-Bold.ttf',26),(G+'Poppins-Light.ttf',64),(G+'Poppins-Light.ttf',22),(G+'Poppins-SemiBold.ttf' if __import__('os').path.exists(G+'Poppins-SemiBold.ttf') else G+'Poppins-Bold.ttf',18)]
chars=[chr(c) for c in range(32,127)]+list("ÄÖÜäöüß€°·–…×•²³„“”‚‘’→✓←↑↓—»«−±")
out=bytearray(b'NFNT'); out+=struct.pack('<I',len(fonts))
for path,size in fonts:
    f=ImageFont.truetype(path,size)
    asc,desc=f.getmetrics()
    out+=struct.pack('<IIII',size,asc+desc,asc,len(chars))
    for ch in chars:
        bbox=f.getbbox(ch)
        adv=f.getlength(ch)
        x0,y0,x1,y1=bbox
        w=max(0,x1-x0); h=max(0,y1-y0)
        data=b''
        if w>0 and h>0:
            im=Image.new('L',(w,h),0)
            d=ImageDraw.Draw(im); d.text((-x0,-y0),ch,font=f,fill=255)
            data=im.tobytes()
        out+=struct.pack('<IhhhHH',ord(ch),int(round(adv)),x0,y0,w,h)+data
open(sys.argv[1],'wb').write(out)
print(len(out))
