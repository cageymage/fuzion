package images

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"net/http"

	"github.com/HugoSmits86/nativewebp"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	fullLongEdge  = 2560
	thumbLongEdge = 800
	storedType    = "image/webp"

	// Rejects decompression bombs: a decoded image costs about 4 bytes per pixel.
	maxSourcePixels = 40_000_000
)

var allowedTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

var errNotAnImage = &ValidationError{Field: "file", Problem: "must be a PNG, JPEG, WebP or GIF image"}

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Upload(ctx context.Context, data []byte, declaredType string) (uuid.UUID, error) {
	src, err := decode(data, declaredType)
	if err != nil {
		return uuid.Nil, err
	}

	fullImage := fitLongEdge(src, fullLongEdge)
	full, err := encodeWebP(fullImage)
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode full image: %w", err)
	}
	thumb, err := encodeWebP(fitLongEdge(src, thumbLongEdge))
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode thumbnail: %w", err)
	}

	bounds := fullImage.Bounds()
	return s.repo.Create(ctx, NewImage{
		ContentType: storedType,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		FullBytes:   full,
		ThumbBytes:  thumb,
	})
}

func (s *Service) Full(ctx context.Context, id string) ([]byte, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.repo.Full(ctx, parsed)
}

func (s *Service) Thumb(ctx context.Context, id string) ([]byte, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.repo.Thumb(ctx, parsed)
}

func decode(data []byte, declaredType string) (image.Image, error) {
	declared, _, err := mime.ParseMediaType(declaredType)
	if err != nil || !allowedTypes[declared] || declared != http.DetectContentType(data) {
		return nil, errNotAnImage
	}

	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 {
		return nil, errNotAnImage
	}
	if config.Width*config.Height > maxSourcePixels {
		return nil, &ValidationError{Field: "file", Problem: "dimensions are too large"}
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errNotAnImage
	}
	return src, nil
}

// An image already within the limit is returned as is; this never upscales.
func fitLongEdge(src image.Image, limit int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	longEdge := max(width, height)
	if longEdge <= limit {
		return src
	}

	dst := image.NewNRGBA(image.Rect(0, 0, max(1, width*limit/longEdge), max(1, height*limit/longEdge)))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

func encodeWebP(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := nativewebp.Encode(&buf, img, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
