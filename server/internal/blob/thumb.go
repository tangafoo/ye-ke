package blob

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // register decoders
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Thumbnail decodes a JPEG/PNG/GIF/WebP, scales it down to at most maxW
// pixels wide (never up), and re-encodes it as JPEG. News photos are stored
// this way: a small credited thumbnail, not the outlet's full image.
func Thumbnail(src []byte, maxW, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("[blob] decode: %w", err)
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return nil, fmt.Errorf("[blob] empty image")
	}
	if w > maxW {
		h = h * maxW / w
		w = maxW
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("[blob] encode: %w", err)
	}
	return out.Bytes(), nil
}
