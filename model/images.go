package model

type OriginalResized struct {
	Original Image
	Resized  Image
}

type Image struct {
	ID          int
	DownloadURL string
	Resolution  string
}

type ImagesRepository interface {
	Save(Image) (int, error)
	All() ([]OriginalResized, error)
	OnlyResized() ([]Image, error)
}
