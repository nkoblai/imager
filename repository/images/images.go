package images

import (
	"database/sql"

	"github.com/imager/model"
)

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{db}
}

func (r Repo) Save(i model.Image) (int, error) {
	return 0, nil
}

func (r Repo) All() ([]model.Image, error) {
	return nil, nil
}

func (r Repo) AllResized() ([]model.Image, error) {
	return nil, nil
}
