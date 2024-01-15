package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

type MinioInstance struct {
	host      string
	accessKey string
	secretKey string
}

type API struct {
	minioAccessor  MinioAccessor
	minioInstances []MinioInstance // slice of instances for sorting purposes (and keeping counter for sorting)
	modulo         int64           // for simplified calculation we store the length of minioInstances
}

func InitAPI() API {
	api := API{}
	api.minioAccessor = InitMinioClient()
	api.SetMinioData()
	return api
}

func (api *API) SetMinioData() {
	instancesInfo, err := api.minioAccessor.getMinioInstancesInfo()
	if err != nil {
		panic(err)
	}
	api.minioInstances = instancesInfo
	api.modulo = int64(len(instancesInfo))
}

func (api *API) serve() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/object/{id}", api.handleObject)
	err := http.ListenAndServe(":3000", myRouter)
	if err != nil {
		fmt.Printf("ERROR: ListenAndServe: %s", err.Error())
		return
	}
}

func (api *API) handleObject(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.handleObjectGet(w, r)
	case http.MethodPost:
		api.handleObjectPost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *API) handleObjectGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectId := vars["id"]

	if api.modulo == 0 {
		// If modulo is equal to zero let's try to determine the instances again - there is a race condition
		api.SetMinioData()
		if api.modulo == 0 {
			panic(errors.New("No Minio instances are identified."))
		}
	}
	index, err := api.moduloFromObjectId(objectId, api.modulo)
	if err != nil {
		http.Error(w, "Error during parsing objectId.", http.StatusBadRequest) // This might be controversial but I believe this is user's fault.
		return
	}
	if index > api.modulo {
		http.Error(w, fmt.Sprintf("Object with id %s not found on any of underlying instances.", objectId), http.StatusBadRequest)
		return
	}
	minioInstance := api.minioInstances[index]
	content, err := api.minioAccessor.getMinioInstanceObject(objectId, minioInstance)
	if err != nil {
		http.Error(w, "Error during fetching object from instance.", http.StatusInternalServerError)
		return
	}
	w.Write(content)
}

func (api *API) handleObjectPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectId := vars["id"]

	if api.modulo == 0 {
		// If modulo is equal to zero let's try to determine the instances again - there is a race condition
		api.SetMinioData()
		if api.modulo == 0 {
			panic(errors.New("No Minio instances are identified."))
		}
	}
	index, err := api.moduloFromObjectId(objectId, api.modulo)
	if err != nil {
		http.Error(w, "Error during parsing objectId.", http.StatusBadRequest) // This might be controversial but I believe this is user's fault.
		return
	}

	// choose one instance based on objectId and number of available instances and save object there
	instance := api.minioInstances[index]
	err = api.minioAccessor.sendContentToMinioInstance(objectId, instance, r.Body, r.ContentLength)
	if err != nil {
		http.Error(w, "Error during sending object to instance.", http.StatusInternalServerError)
	}

	w.Write([]byte("Object saved successfully."))
}

// Get the int64 interpretation of objectId (sum of all the bytes from objectId) and return modulo of that value.
// The value of modulo operand is determined by the number of minio instances and passed to the function.
func (api *API) moduloFromObjectId(objectId string, modulo int64) (int64, error) {
	result := int64(0)
	for _, objectIdRune := range objectId {
		result += int64(objectIdRune)
	}
	return result % modulo, nil
}
