package images

import (
	"database/sql"
	"fmt"

	"github.com/imager/model"
)

const (
	allImagesQuery = "SELECT A.id AS originalID, A.download_url AS original_download_url, A.resolution AS original_resolution, B.id AS resizedID, B.download_url AS resized_download_url, B.resolution AS resized_resolution FROM images A, images B WHERE A.id = B.original_id"

	insertImageWithReferenceQuery    = "INSERT INTO images (download_url, resolution, original_id) VALUES ($1, $2, $3) RETURNING id"
	insertImageWithoutReferenceQuery = "INSERT INTO images (download_url, resolution) VALUES ($1, $2) RETURNING id"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db}
}

func (r Repo) Save(img model.Image) (int, error) {
	const errMsg = "inserting of '%v' to db failed with error: %v"
	var id int
	if img.OriginalID != 0 {
		if err := r.db.QueryRow(insertImageWithReferenceQuery, img.DownloadURL, img.Resolution, img.OriginalID).Scan(&id); err != nil {
			return 0, fmt.Errorf(errMsg, img, err)
		}
		return id, nil
	}
	if err := r.db.QueryRow(insertImageWithoutReferenceQuery, img.DownloadURL, img.Resolution).Scan(&id); err != nil {
		return 0, fmt.Errorf(errMsg, img, err)
	}
	return id, nil
}

func (r Repo) All() ([]model.OriginalResized, error) {
	rows, err := r.db.Query(allImagesQuery)
	if err != nil {
		return nil, fmt.Errorf("error getting all images from DB: %v", err)
	}
	defer rows.Close()

	var res []model.OriginalResized

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
			return nil, err
		}
		res = append(res, originalResized)
	}
	return res, nil
}

func (r Repo) OnlyResized() ([]model.Image, error) {
	return nil, nil
}
