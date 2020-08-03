package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/imager/repository/images"
	"github.com/imager/router"
	"github.com/imager/web/downloader"
	"github.com/imager/web/uploader"
)

func main() {

	// TODO: make it configurable using env vars
	// optional: can be moved to another package
	const (
		host     = "localhost"
		port     = 5432
		user     = "postgres"
		password = "123"
		dbname   = "imager"
	)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("error creating db connection: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// TODO: add possibility to define region using env vars
	session, err := session.NewSession(&aws.Config{Region: aws.String("eu-central-1")})
	if err != nil {
		panic(err)
	}

	s3uploader := s3manager.NewUploader(session)

	if err := http.ListenAndServe(":8080", router.New(images.NewRepo(db), uploader.New(s3uploader), downloader.New())); err != nil {
		panic(err)
	}
}
