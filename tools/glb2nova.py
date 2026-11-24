#!/usr/bin/env python3
"""Konvertiert ausgewählte Kenney-GLB-Modelle (CC0) in das kompakte Nova-Format.

Ausgabe (gzip, little endian):
  u32 Anzahl Modelle
  je Modell: u16 Namenslänge, Name, u32 nv, u32 ni,
             nv * (f32 px,py,pz, i8 nx,ny,nz, u8 r,g,b, u8 mat), ni * u32
Farben werden aus der colormap-Textur (UV) oder baseColorFactor gebacken.
"""
import gzip
import json
import math
import os
import struct
import sys

from PIL import Image

MAT_MATTE, MAT_PANEL, MAT_GLASS, MAT_EMIT, MAT_LEAF, MAT_SOLAR, MAT_METAL = 0, 1, 2, 3, 4, 5, 7

COMP = {5120: ('b', 1), 5121: ('B', 1), 5122: ('h', 2), 5123: ('H', 2), 5125: ('I', 4), 5126: ('f', 4)}
NCOMP = {'SCALAR': 1, 'VEC2': 2, 'VEC3': 3, 'VEC4': 4, 'MAT4': 16}

_tex_cache = {}


def load_glb(path):
    d = open(path, 'rb').read()
    assert d[:4] == b'glTF'
    off = 12
    js, binc = None, None
    while off < len(d):
        ln, typ = struct.unpack('<II', d[off:off + 8])
        chunk = d[off + 8:off + 8 + ln]
        if typ == 0x4E4F534A:
            js = json.loads(chunk)
        elif typ == 0x004E4942:
            binc = chunk
        off += 8 + ln
    return js, binc


def accessor(js, binc, idx):
    a = js['accessors'][idx]
    bv = js['bufferViews'][a['bufferView']]
    fmt, sz = COMP[a['componentType']]
    nc = NCOMP[a['type']]
    stride = bv.get('byteStride', sz * nc)
    base = bv.get('byteOffset', 0) + a.get('byteOffset', 0)
    out = []
    for i in range(a['count']):
        p = base + i * stride
        v = struct.unpack('<' + fmt * nc, binc[p:p + sz * nc])
        if a.get('normalized'):
            m = {1: 127.0, 2: 32767.0}[sz] if fmt.islower() else {1: 255.0, 2: 65535.0}[sz]
            v = tuple(x / m for x in v)
        out.append(v)
    return out


def qmat(t, r, s):
    x, y, z, w = r
    m = [
        [1 - 2 * (y * y + z * z), 2 * (x * y - z * w), 2 * (x * z + y * w), 0],
        [2 * (x * y + z * w), 1 - 2 * (x * x + z * z), 2 * (y * z - x * w), 0],
        [2 * (x * z - y * w), 2 * (y * z + x * w), 1 - 2 * (x * x + y * y), 0],
        [0, 0, 0, 1],
    ]
    for i in range(3):
        for j in range(3):
            m[i][j] *= s[j]
        m[i][3] = t[i]
    return m


def mmul(a, b):
    return [[sum(a[i][k] * b[k][j] for k in range(4)) for j in range(4)] for i in range(4)]


def node_mat(n):
    if 'matrix' in n:
        m = n['matrix']
        return [[m[c * 4 + r] for c in range(4)] for r in range(4)]
    return qmat(n.get('translation', [0, 0, 0]), n.get('rotation', [0, 0, 0, 1]), n.get('scale', [1, 1, 1]))


def srgb(c):
    c = max(0.0, min(1.0, c))
    return c ** (1 / 2.2)


def classify(rgb, name):
    r, g, b = rgb
    nm = (name or '').lower()
    if 'glass' in nm or 'window' in nm:
        return MAT_GLASS
    if 'light' in nm or 'emis' in nm or 'neon' in nm:
        return MAT_EMIT
    if 'metal' in nm:
        return MAT_METAL
    # helle Blautöne in der colormap = Fenster
    if b > 150 and b > r + 35 and b > g + 10:
        return MAT_GLASS
    # helle blau-lila Scheiben der Stadt-Sets
    if (r + g + b) / 3 > 165 and b - r > 15 and b < 245:
        return MAT_GLASS
    if r < 60 and g < 60 and b < 70:
        return MAT_MATTE
    return MAT_PANEL


def convert(path):
    js, binc = load_glb(path)
    tex = None
    for m in js.get('materials', []):
        if 'baseColorTexture' in m.get('pbrMetallicRoughness', {}):
            img = js['images'][0]['uri']
            tp = os.path.join(os.path.dirname(path), img)
            if tp not in _tex_cache:
                _tex_cache[tp] = Image.open(tp).convert('RGB')
            tex = _tex_cache[tp]
    V, I = [], []
    scene = js['scenes'][js.get('scene', 0)]

    def walk(ni, parent):
        n = js['nodes'][ni]
        M = mmul(parent, node_mat(n))
        if 'mesh' in n:
            for prim in js['meshes'][n['mesh']]['primitives']:
                at = prim['attributes']
                P = accessor(js, binc, at['POSITION'])
                N = accessor(js, binc, at['NORMAL']) if 'NORMAL' in at else [(0, 1, 0)] * len(P)
                UV = accessor(js, binc, at['TEXCOORD_0']) if 'TEXCOORD_0' in at else None
                mat = js['materials'][prim['material']] if 'material' in prim else {}
                pbr = mat.get('pbrMetallicRoughness', {})
                bcf = pbr.get('baseColorFactor', [1, 1, 1, 1])
                em = mat.get('emissiveFactor', [0, 0, 0])
                base = len(V)
                for k, p in enumerate(P):
                    x = M[0][0] * p[0] + M[0][1] * p[1] + M[0][2] * p[2] + M[0][3]
                    y = M[1][0] * p[0] + M[1][1] * p[1] + M[1][2] * p[2] + M[1][3]
                    z = M[2][0] * p[0] + M[2][1] * p[1] + M[2][2] * p[2] + M[2][3]
                    nn = N[k]
                    nx = M[0][0] * nn[0] + M[0][1] * nn[1] + M[0][2] * nn[2]
                    ny = M[1][0] * nn[0] + M[1][1] * nn[1] + M[1][2] * nn[2]
                    nz = M[2][0] * nn[0] + M[2][1] * nn[1] + M[2][2] * nn[2]
                    ln = math.sqrt(nx * nx + ny * ny + nz * nz) or 1
                    if tex is not None and UV is not None and 'baseColorTexture' in pbr:
                        u, v = UV[k]
                        px = int((u % 1.0) * tex.width) % tex.width
                        py = int((v % 1.0) * tex.height) % tex.height
                        rgb = tex.getpixel((px, py))
                    else:
                        rgb = tuple(int(srgb(c) * 255) for c in bcf[:3])
                    mt = classify(rgb, mat.get('name'))
                    if max(em) > 0.3:
                        mt = MAT_EMIT
                    V.append((x, y, z, nx / ln, ny / ln, nz / ln, rgb, mt))
                if 'indices' in prim:
                    idx = [i[0] for i in accessor(js, binc, prim['indices'])]
                else:
                    idx = list(range(len(P)))
                I.extend(base + i for i in idx)
        for c in n.get('children', []):
            walk(c, M)

    ident = [[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 1, 0], [0, 0, 0, 1]]
    for ni in scene['nodes']:
        walk(ni, ident)
    return V, I


def main():
    listfile, out = sys.argv[1], sys.argv[2]
    entries = []
    for line in open(listfile):
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        key, path = line.split('=', 1)
        entries.append((key.strip(), path.strip()))
    buf = bytearray(struct.pack('<I', len(entries)))
    total = 0
    for key, path in entries:
        V, I = convert(path)
        nb = key.encode()
        buf += struct.pack('<H', len(nb)) + nb + struct.pack('<II', len(V), len(I))
        for x, y, z, nx, ny, nz, rgb, mt in V:
            buf += struct.pack('<fffbbbBBBB', x, y, z, int(nx * 127), int(ny * 127), int(nz * 127), rgb[0], rgb[1], rgb[2], mt)
        buf += struct.pack('<%dI' % len(I), *I)
        total += len(I) // 3
        print('%-22s %6d tris  %s' % (key, len(I) // 3, os.path.basename(path)))
    with gzip.open(out, 'wb', 9) as f:
        f.write(bytes(buf))
    print('gesamt', total, 'Dreiecke,', os.path.getsize(out), 'Bytes')


if __name__ == '__main__':
    main()
