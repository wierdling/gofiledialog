package gofiledialog

import (
	"bytes"
	"container/heap"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"fyne.io/fyne/v2"

	"github.com/wierdling/gofiledialog/internal/lru"
)

// errImageTooLarge is returned (and simply treated as "no thumbnail, keep
// the file-type icon") when an image is too large to be worth the CPU/memory
// cost of decoding just for a small preview. A single large decode can take
// seconds and gigabytes of memory (e.g. a 48-megapixel photo takes over a
// second to JPEG-decode in pure Go), which is wasteful — and with several
// such files visible at once in an icon view, adds up to a very sluggish UI.
var errImageTooLarge = errors.New("image too large to thumbnail")

// Files larger than this are skipped without even reading them, since size
// is already known from the directory listing.
const maxThumbnailFileBytes = 25 * 1024 * 1024 // 25 MB

// Images whose pixel count exceeds this are skipped after a cheap
// header-only peek (image.DecodeConfig), before paying for a full decode.
const maxThumbnailPixels = 24_000_000 // ~24 MP, e.g. 6000x4000

var thumbnailableExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".tif": true, ".tiff": true, ".webp": true,
}

func isThumbnailable(name string) bool {
	return thumbnailableExts[strings.ToLower(filepath.Ext(name))]
}

const thumbnailQueueSize = 256
const thumbnailMemoryEntries = 200
const maxThumbnailWorkers = 8

// thumbnailer generates image-file thumbnails off the UI goroutine, with a
// bounded worker pool plus in-memory and on-disk caches so revisiting a
// folder is instant. Pending requests are served smallest-file-first (see
// jobQueue), so a folder mixing many small icons with a few huge photos
// shows most of its thumbnails quickly instead of stalling behind the big
// ones in arrival order.
type thumbnailer struct {
	mu         sync.Mutex
	cond       *sync.Cond
	queue      jobQueue
	memory     *lru.Cache
	diskDir    string
	inflight   map[string]*thumbnailRequest
	generation uint64
}

type thumbJob struct {
	entry      FileEntry
	size       int
	key        string
	generation uint64
}

type thumbnailRequest struct {
	generation uint64
	done       []func(fyne.Resource)
}

// jobQueue is a container/heap min-heap ordered by file size, smallest first.
type jobQueue []thumbJob

func (q jobQueue) Len() int           { return len(q) }
func (q jobQueue) Less(i, j int) bool { return q[i].entry.Size < q[j].entry.Size }
func (q jobQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *jobQueue) Push(x any)        { *q = append(*q, x.(thumbJob)) }
func (q *jobQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	*q = old[:n-1]
	return item
}

func newThumbnailer() *thumbnailer {
	t := &thumbnailer{
		memory:   lru.New(thumbnailMemoryEntries),
		inflight: make(map[string]*thumbnailRequest),
	}
	t.cond = sync.NewCond(&t.mu)
	if dir, err := os.UserCacheDir(); err == nil {
		t.diskDir = filepath.Join(dir, "wierdling-gofiledialog", "thumbcache")
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > maxThumbnailWorkers {
		workers = maxThumbnailWorkers
	}
	for i := 0; i < workers; i++ {
		go t.worker()
	}
	return t
}

func (t *thumbnailer) worker() {
	for {
		t.mu.Lock()
		for t.queue.Len() == 0 {
			t.cond.Wait()
		}
		job := heap.Pop(&t.queue).(thumbJob)
		t.mu.Unlock()

		res := t.generate(job.entry, job.size)
		t.finish(job.key, job.generation, res)
	}
}

func (t *thumbnailer) finish(key string, generation uint64, res fyne.Resource) {
	t.mu.Lock()
	request, ok := t.inflight[key]
	if !ok || request.generation != generation {
		t.mu.Unlock()
		return
	}
	delete(t.inflight, key)
	done := append([]func(fyne.Resource){}, request.done...)
	t.mu.Unlock()

	fyne.Do(func() {
		for _, callback := range done {
			callback(res)
		}
	})
}

// CancelPending drops queued work from an old directory. A worker that is
// already decoding one file is allowed to finish, but its result is ignored
// when it belongs to the previous generation.
func (t *thumbnailer) CancelPending() {
	t.mu.Lock()
	t.generation++
	t.queue = nil
	t.inflight = make(map[string]*thumbnailRequest)
	t.mu.Unlock()
}

// Request queues async thumbnail generation for entry at the given pixel
// size. For requests that remain current, done is called exactly once, on the
// Fyne UI goroutine — with a real resource on success, or nil if entry isn't
// a (small enough) image, decoding failed, or the queue was full. Requests
// canceled by a directory change are dropped because their cells are stale.
// Callers showing a "loading" placeholder can therefore always clear it for
// current requests, regardless of outcome.
func (t *thumbnailer) Request(entry FileEntry, size int, done func(fyne.Resource)) {
	if done == nil {
		return
	}
	if entry.IsDir || !isThumbnailable(entry.Name) || entry.Size > maxThumbnailFileBytes {
		done(nil)
		return
	}
	key := thumbnailCacheKey(entry, size)
	if cached, ok := t.memory.Get(key); ok {
		done(cached.(fyne.Resource))
		return
	}

	t.mu.Lock()
	if request, ok := t.inflight[key]; ok {
		request.done = append(request.done, done)
		t.mu.Unlock()
		return
	}
	if t.queue.Len() >= thumbnailQueueSize {
		t.mu.Unlock()
		done(nil)
		return
	}
	generation := t.generation
	t.inflight[key] = &thumbnailRequest{generation: generation, done: []func(fyne.Resource){done}}
	heap.Push(&t.queue, thumbJob{entry: entry, size: size, key: key, generation: generation})
	t.mu.Unlock()
	t.cond.Signal()
}

// MightThumbnail reports whether Request is likely to attempt a real decode
// for entry (as opposed to resolving immediately with nil), so a caller can
// decide whether showing a "loading" placeholder is worthwhile. It's a
// best-effort hint, not a guarantee — Request still applies the same rules.
func (t *thumbnailer) MightThumbnail(entry FileEntry) bool {
	return !entry.IsDir && isThumbnailable(entry.Name) && entry.Size <= maxThumbnailFileBytes
}

func thumbnailCacheKey(entry FileEntry, size int) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%d|%d|%d", entry.Path, entry.Size, entry.ModTime.UnixNano(), size)
	return hex.EncodeToString(h.Sum(nil))
}

func (t *thumbnailer) generate(entry FileEntry, size int) fyne.Resource {
	key := thumbnailCacheKey(entry, size)
	if cached, ok := t.memory.Get(key); ok {
		return cached.(fyne.Resource)
	}

	diskPath := ""
	if t.diskDir != "" {
		diskPath = filepath.Join(t.diskDir, key+".png")
		if data, err := os.ReadFile(diskPath); err == nil {
			res := fyne.NewStaticResource(key, data)
			t.memory.Add(key, res)
			return res
		}
	}

	data, err := renderThumbnail(entry.Path, size)
	if err != nil {
		return nil
	}
	res := fyne.NewStaticResource(key, data)
	t.memory.Add(key, res)
	if diskPath != "" && os.MkdirAll(t.diskDir, 0o755) == nil {
		_ = os.WriteFile(diskPath, data, 0o644)
	}
	return res
}

// renderThumbnail decodes the image at path and downscales it to fit within
// a size x size box (never upscaling), returning PNG-encoded bytes.
func renderThumbnail(path string, size int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	if cfg.Width*cfg.Height > maxThumbnailPixels {
		return nil, errImageTooLarge
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	longest := w
	if h > longest {
		longest = h
	}
	scale := float64(size) / float64(longest)
	if scale > 1 {
		scale = 1
	}
	dstW, dstH := int(float64(w)*scale), int(float64(h)*scale)
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
