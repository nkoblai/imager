package images

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/imager/src/model"
)

const (
	allImagesQuery = `SELECT
	 A.id AS originalID,
	 A.download_url AS original_download_url, 
	 A.resolution AS original_resolution, 
	 B.id AS resizedID, 
	 B.download_url AS resized_download_url, 
	 B.resolution AS resized_resolution 
	 FROM images A, images B WHERE A.id = B.original_id`

	onlyResizedImagesQuery           = "SELECT id, download_url, resolution FROM images WHERE original_id IS NOT NULL"
	oneByID                          = "SELECT id, download_url, resolution FROM images WHERE id = $1"
	insertImageWithReferenceQuery    = "INSERT INTO images (download_url, resolution, original_id) VALUES ($1, $2, $3) RETURNING id"
	insertImageWithoutReferenceQuery = "INSERT INTO images (download_url, resolution) VALUES ($1, $2) RETURNING id"
)

// Repo contains db session.
type Repo struct {
	db *sql.DB
}

// NewRepo creates new Repo struct with db session.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db}
}

// Save inserts new image with or without reference.
func (r *Repo) Save(ctx context.Context, img model.Image) (int, error) {
	const errMsg = "inserting of '%v' to db failed with error: %v"
	var id int
	if img.OriginalID != 0 {
		if err := r.db.QueryRowContext(ctx, insertImageWithReferenceQuery, img.DownloadURL, img.Resolution, img.OriginalID).Scan(&id); err != nil {
			return 0, fmt.Errorf(errMsg, img, err)
		}
		return id, nil
	}
	if err := r.db.QueryRowContext(ctx, insertImageWithoutReferenceQuery, img.DownloadURL, img.Resolution).Scan(&id); err != nil {
		return 0, fmt.Errorf(errMsg, img, err)
	}
	return id, nil
}

// All returns all images.
func (r *Repo) All(ctx context.Context) ([]model.OriginalResized, error) {
	const errMsg = "error getting all images from DB: %v"
	rows, err := r.db.QueryContext(ctx, allImagesQuery)
	if err != nil {
		return nil, fmt.Errorf(errMsg, err)
	}
	defer rows.Close()

	res := []model.OriginalResized{}
	for rows.Next() {
		var originalResized model.OriginalResized
		if err := rows.Scan(
			&originalResized.Original.ID,
			&originalResized.Original.DownloadURL,
			&originalResized.Original.Resolution,
			&originalResized.Resized.ID,
			&originalResized.Resized.DownloadURL,
			&originalResized.Resized.Resolution,
		); err != nil {
			return nil, fmt.Errorf(errMsg, err)
		}
		res = append(res, originalResized)
	}
	return res, nil
}

// OnlyResized returns only resized images.
func (r *Repo) OnlyResized(ctx context.Context) ([]model.Image, error) {
	const errMsg = "error getting only resized images from DB: %v"
	rows, err := r.db.QueryContext(ctx, onlyResizedImagesQuery)
	if err != nil {
		return nil, fmt.Errorf(errMsg, err)
	}
	defer rows.Close()

	res := []model.Image{}
	for rows.Next() {
		var image model.Image
		if err := rows.Scan(
			&image.ID,
			&image.DownloadURL,
			&image.Resolution,
		); err != nil {
			return nil, fmt.Errorf(errMsg, err)
		}
		res = append(res, image)
	}
	return res, nil
}

// GetOne returns specific image by it's ID.
func (r *Repo) GetOne(ctx context.Context, id int) (model.Image, error) {
	var image model.Image
	if err := r.db.QueryRowContext(ctx, oneByID, id).Scan(&image.ID, &image.DownloadURL, &image.Resolution); err != nil {
		return model.Image{}, fmt.Errorf("error getting image by ID: %d, error: %v", id, err)
	}
	return image, nil
}
