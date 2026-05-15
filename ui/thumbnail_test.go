package ui

import (
	"image"
	"image/color"
	"testing"
)

func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestScaleContain_OutputDimensions(t *testing.T) {
	src := solidImage(200, 100, color.White)
	dst := scaleContain(src, 100, 100)
	b := dst.Bounds()
	if b.Dx() != 100 || b.Dy() != 100 {
		t.Errorf("expected 100x100 output, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestScaleContain_WideImageFitsWidth(t *testing.T) {
	// 200x100 source scaled into 100x100 box: width fills 100, height = 50.
	src := solidImage(200, 100, color.White)
	dst := scaleContain(src, 100, 100)

	// The center row should be opaque (image pixels).
	cx, cy := 50, 50
	_, _, _, a := dst.At(cx, cy).RGBA()
	if a == 0 {
		t.Error("expected center pixel to be opaque for wide image")
	}
	// Top row should be transparent (letterbox padding).
	_, _, _, aTop := dst.At(50, 0).RGBA()
	if aTop != 0 {
		t.Error("expected top row to be transparent (letterbox) for wide image")
	}
}

func TestScaleContain_TallImageFitsHeight(t *testing.T) {
	// 100x200 source scaled into 100x100 box: height fills 100, width = 50.
	src := solidImage(100, 200, color.White)
	dst := scaleContain(src, 100, 100)

	// Center column should be opaque.
	_, _, _, a := dst.At(50, 50).RGBA()
	if a == 0 {
		t.Error("expected center pixel to be opaque for tall image")
	}
	// Left column should be transparent (pillarbox padding).
	_, _, _, aLeft := dst.At(0, 50).RGBA()
	if aLeft != 0 {
		t.Error("expected left column to be transparent (pillarbox) for tall image")
	}
}

func TestScaleContain_SquareImageFillsBox(t *testing.T) {
	src := solidImage(200, 200, color.White)
	dst := scaleContain(src, 100, 100)

	// All corners should be opaque — no padding needed.
	corners := [][2]int{{0, 0}, {99, 0}, {0, 99}, {99, 99}}
	for _, c := range corners {
		_, _, _, a := dst.At(c[0], c[1]).RGBA()
		if a == 0 {
			t.Errorf("expected corner (%d,%d) to be opaque for square image", c[0], c[1])
		}
	}
}

func TestScaleContain_NonSquareBox(t *testing.T) {
	// 1920x1016 (typical screenshot) into 100x80 box.
	src := solidImage(1920, 1016, color.White)
	dst := scaleContain(src, 100, 80)
	b := dst.Bounds()
	if b.Dx() != 100 || b.Dy() != 80 {
		t.Errorf("expected 100x80 output, got %dx%d", b.Dx(), b.Dy())
	}
	// Center should be opaque.
	_, _, _, a := dst.At(50, 40).RGBA()
	if a == 0 {
		t.Error("expected center pixel to be opaque")
	}
}
