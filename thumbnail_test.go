package gofiledialog

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"fyne.io/fyne/v2"
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

func TestRenderThumbnailRejectsMalformedImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.png")
	if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := renderThumbnail(path, 20); err == nil {
		t.Fatal("expected malformed image to fail")
	}
}

func TestRenderThumbnailRejectsOversizedImageFromHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.bmp")
	if err := os.WriteFile(path, bmpHeader(6000, 5000), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := renderThumbnail(path, 20); !errors.Is(err, errImageTooLarge) {
		t.Fatalf("err = %v, want %v", err, errImageTooLarge)
	}
}

func TestThumbnailerCloseStopsIdleWorkers(t *testing.T) {
	baseline := runtime.NumGoroutine()
	thumbs := newThumbnailerWithWorkers(3)
	waitForGoroutinesAtLeast(t, baseline+3)

	thumbs.Close()
	waitForGoroutinesAtMost(t, baseline+1)
}

func bmpHeader(width, height int32) []byte {
	data := make([]byte, 54)
	copy(data[0:2], []byte{'B', 'M'})
	binary.LittleEndian.PutUint32(data[2:6], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[10:14], 54)
	binary.LittleEndian.PutUint32(data[14:18], 40)
	binary.LittleEndian.PutUint32(data[18:22], uint32(width))
	binary.LittleEndian.PutUint32(data[22:26], uint32(height))
	binary.LittleEndian.PutUint16(data[26:28], 1)
	binary.LittleEndian.PutUint16(data[28:30], 24)
	return data
}

func TestThumbnailerCloseSuppressesActiveCallback(t *testing.T) {
	thumbs := newThumbnailerWithWorkers(1)
	defer thumbs.Close()

	started := make(chan struct{})
	release := make(chan struct{})
	thumbs.generate = func(FileEntry, int) fyne.Resource {
		close(started)
		<-release
		return fyne.NewStaticResource("late.png", []byte("late"))
	}

	called := make(chan struct{}, 1)
	thumbs.Request(FileEntry{
		Name:    "source.png",
		Path:    filepath.Join(t.TempDir(), "source.png"),
		Size:    100,
		ModTime: time.Now(),
	}, 32, func(fyne.Resource) {
		called <- struct{}{}
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("thumbnail worker did not start")
	}

	closed := make(chan struct{})
	go func() {
		thumbs.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned before active worker finished")
	case <-time.After(10 * time.Millisecond):
	}

	close(release)

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after active worker finished")
	}
	select {
	case <-called:
		t.Fatal("callback ran after thumbnailer was closed")
	default:
	}
}

func TestThumbnailerRequestAfterCloseIsIgnored(t *testing.T) {
	thumbs := newThumbnailerWithWorkers(1)
	thumbs.Close()

	called := false
	thumbs.Request(FileEntry{Name: "source.png", Size: 100}, 32, func(fyne.Resource) {
		called = true
	})
	if called {
		t.Fatal("request callback ran after thumbnailer was closed")
	}
	if thumbs.MightThumbnail(FileEntry{Name: "source.png", Size: 100}) {
		t.Fatal("closed thumbnailer should not report entries as thumbnailable")
	}
}

func waitForGoroutinesAtLeast(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("goroutines = %d, want at least %d", runtime.NumGoroutine(), want)
}

func waitForGoroutinesAtMost(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("goroutines = %d, want at most %d", runtime.NumGoroutine(), want)
}
