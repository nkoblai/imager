package model

import "context"

// OriginalResized describes original and changed images.
type OriginalResized struct {
	Original Image
	Resized  Image
}

// Image describes image.
type Image struct {
	ID          int
	DownloadURL string
	Resolution  string
	OriginalID  int `json:",omitempty"`
}

// ImagesRepository describes methods for working with DB.
type ImagesRepository interface {
	Save(context.Context, Image) (int, error)
	All(context.Context) ([]OriginalResized, error)
	OnlyResized(context.Context) ([]Image, error)
	GetOne(context.Context, int) (Image, error)
}
