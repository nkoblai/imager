package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/imager/src/repository/images"
	"github.com/imager/src/router"
	"github.com/imager/src/web/downloader"
	"github.com/imager/src/web/uploader"
)

func main() {
	db, err := createDBsession()
	if err != nil {
		log.Fatalf("error creating db connection: %v\n", err)
	}
	defer db.Close()

	session, err := session.NewSession()
	if err != nil {
		log.Fatalf("error creating aws session: %v\n", err)
	}

	s3uploader := s3manager.NewUploader(session)

	if err := http.ListenAndServe(":8080", router.New(images.NewRepo(db), uploader.New(s3uploader), downloader.New())); err != nil {
		log.Fatalf("error running server: %v\n", err)
	}
}

func createDBsession() (*sql.DB, error) {
	host := os.Getenv("PGHOST")
	port := os.Getenv("PGPORT")
	user := os.Getenv("PGUSER")
	password := os.Getenv("PGPASSWORD")
	dbname := os.Getenv("PGDBNAME")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	return sql.Open("postgres", psqlInfo)
}
