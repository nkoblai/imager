package model

import "context"

type OriginalResized struct {
	Original Image
	Resized  Image
}

type Image struct {
	ID          int
	DownloadURL string
	Resolution  string
	OriginalID  int `json:",omitempty"`
}

type ImagesRepository interface {
	Save(context.Context, Image) (int, error)
	All(context.Context) ([]OriginalResized, error)
	OnlyResized(context.Context) ([]Image, error)
}
