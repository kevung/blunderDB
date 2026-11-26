#!/usr/bin/env python3
"""Generate JPEG 2000 test images and PNG reference decodes.

Requires Pillow built with OpenJPEG support:
    pip install pillow
Run from the repository root:
    python3 testdata/gen.py

Each generated .j2k/.jp2 file gets a matching .png containing the pixels
that OpenJPEG (via Pillow) decodes from it. Tests compare this library's
output against those references: bit-exact for lossless, within a small
tolerance for lossy.
"""

import os

import numpy as np
from PIL import Image

HERE = os.path.dirname(os.path.abspath(__file__))


def gradient_rgb(w, h):
    y, x = np.mgrid[0:h, 0:w]
    r = (x * 255 // max(w - 1, 1)).astype(np.uint8)
    g = (y * 255 // max(h - 1, 1)).astype(np.uint8)
    b = ((x + y) * 255 // max(w + h - 2, 1)).astype(np.uint8)
    rng = np.random.default_rng(42)
    noise = rng.integers(0, 32, size=(h, w), dtype=np.uint8)
    return np.stack([r, g | noise, b ^ noise], axis=-1)


def save(name, img, **params):
    path = os.path.join(HERE, name)
    img.save(path, **params)
    ref = Image.open(path)
    ref.load()
    ref.save(os.path.join(HERE, os.path.splitext(name)[0] + ".png"))
    print(f"{name}: {ref.size} {ref.mode} {os.path.getsize(path)}B")


def main():
    rgb = Image.fromarray(gradient_rgb(97, 61), "RGB")
    gray = rgb.convert("L")

    # Lossless (reversible 5/3), raw codestream and JP2 container.
    save("rgb_lossless.j2k", rgb, irreversible=False)
    save("rgb_lossless.jp2", rgb, irreversible=False)
    save("gray_lossless.j2k", gray, irreversible=False)

    # Lossy (irreversible 9/7).
    save("rgb_lossy.j2k", rgb, irreversible=True,
         quality_mode="dB", quality_layers=[45])
    save("gray_lossy.j2k", gray, irreversible=True,
         quality_mode="dB", quality_layers=[45])

    # Multiple quality layers.
    save("rgb_layers.j2k", rgb, irreversible=True,
         quality_mode="dB", quality_layers=[30, 40, 50])

    # Multi-tile.
    big = Image.fromarray(gradient_rgb(200, 131), "RGB")
    save("rgb_tiled.j2k", big, irreversible=False, tile_size=(64, 64))

    # Small images (DWT edge cases, more levels than samples).
    save("tiny_3x5.j2k", Image.fromarray(gradient_rgb(3, 5), "RGB"),
         irreversible=False)
    save("tiny_1x7.j2k", Image.fromarray(gradient_rgb(1, 7), "RGB"),
         irreversible=False)

    # Non-default coding parameters.
    save("rgb_cb32.j2k", rgb, irreversible=False, codeblock_size=(32, 32))
    save("rgb_precincts.j2k", rgb, irreversible=False,
         precinct_size=(128, 128), codeblock_size=(32, 32))
    save("rgb_res3.j2k", rgb, irreversible=False, num_resolutions=3)
    for order in ("RLCP", "RPCL", "PCRL", "CPRL"):
        save(f"rgb_{order.lower()}.j2k", rgb, irreversible=False,
             progression=order)
    save("rgb_nomct.j2k", rgb, irreversible=False, mct=0)

    # RGBA.
    a = np.array(gradient_rgb(64, 48))
    alpha = np.linspace(0, 255, 64, dtype=np.uint8)[None, :].repeat(48, 0)
    rgba = Image.fromarray(np.dstack([a, alpha]), "RGBA")
    save("rgba_lossless.jp2", rgba, irreversible=False)


if __name__ == "__main__":
    main()
