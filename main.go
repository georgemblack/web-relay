package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

// Constants
const (
	WebCertSecretName = "projects/871975485367/secrets/web-origin-cert/versions/1"
	WebKeySecretName  = "projects/871975485367/secrets/web-origin-key/versions/1"
)

func main() {
	port := getEnv("PORT", "8080")

	// pull secrets & write to file system
	certFile, err := os.Create("cert.crt")
	handleErr(err)
	keyFile, err := os.Create("key.key")
	handleErr(err)
	cert, err := getSecret(WebCertSecretName)
	handleErr(err)
	key, err := getSecret(WebKeySecretName)
	handleErr(err)
	_, err = certFile.Write(cert)
	handleErr(err)
	_, err = keyFile.Write(key)
	handleErr(err)

	context := context.Background()
	client, err := storage.NewClient(context)
	handleErr(err)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		key := getObjectKey(r.URL.Path)
		reader, err := client.Bucket("george.black").Object(key).NewReader(context)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		io.Copy(w, reader)
	})

	log.Println("Listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getSecret(name string) ([]byte, error) {
	context := context.Background()
	client, err := secretmanager.NewClient(context)
	if err != nil {
		return nil, err
	}
	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	}
	result, err := client.AccessSecretVersion(context, request)
	if err != nil {
		return nil, err
	}
	return result.GetPayload().GetData(), nil
}

func getObjectKey(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "index.html"
	}
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	if !strings.Contains(path, ".") {
		path = path + "/index.html"
	}
	return path
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func handleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
