package gofiledialog

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsThumbnailableIncludesStageFiveFormats(t *testing.T) {
	for _, name := range []string{"photo.png", "photo.jpg", "photo.jpeg", "photo.gif", "photo.bmp", "photo.tif", "photo.tiff", "photo.webp"} {
		if !isThumbnailable(name) {
			t.Fatalf("%s should be thumbnailable", name)
		}
	}
	if isThumbnailable("vector.svg") {
		t.Fatal("svg should use the image file icon, not the raster thumbnail pipeline")
	}
}

func TestThumbnailCacheKeyChangesWhenFileIdentityChanges(t *testing.T) {
	entry := FileEntry{
		Path:    filepath.Join("C:", "photos", "image.png"),
		Size:    100,
		ModTime: time.Unix(10, 20),
	}

	base := thumbnailCacheKey(entry, 96)
	entry.Size++
	if got := thumbnailCacheKey(entry, 96); got == base {
		t.Fatal("cache key did not change after file size changed")
	}
	entry.Size--
	entry.ModTime = entry.ModTime.Add(time.Nanosecond)
	if got := thumbnailCacheKey(entry, 96); got == base {
		t.Fatal("cache key did not change after mod time changed")
	}
	if got := thumbnailCacheKey(entry, 160); got == base {
		t.Fatal("cache key did not change after thumbnail size changed")
	}
}

func TestRenderThumbnailDownscalesImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.png")
	src := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, src); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := renderThumbnail(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	thumb, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if thumb.Width != 20 || thumb.Height != 10 {
		t.Fatalf("thumbnail size = %dx%d, want 20x10", thumb.Width, thumb.Height)
	}
}
