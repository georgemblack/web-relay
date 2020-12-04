package main

import (
	"context"
	"errors"
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
	err := initCerts()
	if err != nil {
		panic(err)
	}

	context := context.Background()
	client, err := storage.NewClient(context)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Access-Control-Allow-Origin", "https://george.black")
		res.Header().Add("Access-Control-Allow-Methods", "GET, OPTIONS")

		if req.Method == "OPTIONS" {
			return
		}
		if req.Method != "GET" {
			http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		key := getObjectKey(req.URL.Path)
		reader, err := client.Bucket(req.Host).Object(key).NewReader(context)
		if err != nil {
			if errors.Is(err, storage.ErrObjectNotExist) {
				http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		res.Header().Add("Cache-Control", "public, max-age="+getCacheMaxAge(key))
		_, err = io.Copy(res, reader)
		if err != nil {
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Listening on 443")
	log.Fatal(http.ListenAndServeTLS(":443", "cert.crt", "key.key", nil))
}

func initCerts() error {
	certFile, err := os.Create("cert.crt")
	if err != nil {
		return err
	}
	keyFile, err := os.Create("key.key")
	if err != nil {
		return err
	}
	cert, err := getSecret(WebCertSecretName)
	if err != nil {
		return err
	}
	key, err := getSecret(WebKeySecretName)
	if err != nil {
		return err
	}
	_, err = certFile.Write(cert)
	if err != nil {
		return err
	}
	_, err = keyFile.Write(key)
	if err != nil {
		return err
	}
	return nil
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

func getCacheMaxAge(key string) string {
	split := strings.Split(key, ".")
	extension := split[len(split)-1]
	for _, ext := range [...]string{"html", "xml", "json", "txt"} {
		if extension == ext {
			return "900"
		}
	}
	for _, ext := range [...]string{"js", "css"} {
		if extension == ext {
			return "172800"
		}
	}

	return "2592000"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
