package main

import (
	"database/sql"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/imager/repository/images"
	"github.com/imager/router"
	"github.com/imager/uploader"
)

func main() {
	// db, err := sql.Open("", "")
	// if err != nil {
	// 	log.Fatalf("error creating db connection: %v\n", err)
	// 	os.Exit(1)
	// }
	db := &sql.DB{}
	session, err := session.NewSession(&aws.Config{Region: aws.String("eu-central-1")})
	if err != nil {
		panic(err)
	}

	s3uploader := s3manager.NewUploader(session)

	if err := http.ListenAndServe(":8080", router.New(images.NewRepo(db), uploader.New(s3uploader))); err != nil {
		panic(err)
	}
}
