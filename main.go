package main

func main() {
	// initialize API with minio client underneath
  api := InitAPI()
	api.serve()
}
