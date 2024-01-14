package main

import (
	"fmt"
	"net/http"
  "sort"

	"github.com/gorilla/mux"
)

type MinioInstance struct {
	host      string
	accessKey string
	secretKey string
	// The one below can be improved - on the initialization of API we can ask minio instances on the amount
	// of files they have in buckets and use them as offsets here. The name should be changed in that case.
	// Likewise, on init we can add fetch all from minio instances to fill the objectMap with existing objects.
	filesPushedThisRuntime int
}

type API struct {
	minioAccessor    MinioAccessor
	minioInstances []MinioInstance // slice of instances for sorting purposes (and keeping counter for sorting)
	objectMap      map[string]MinioInstance
}

func InitAPI() API {
	api := API{}
	api.minioAccessor = InitMinioClient()
	instancesInfo, err := api.minioAccessor.getMinioInstancesInfo()
	if err != nil {
		panic(err)
	}
	api.minioInstances = instancesInfo
	api.objectMap = make(map[string]MinioInstance)
	return api
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
	// get the instance from id and access it. If the instance is unavailable (important to learn how to determine this from error message) - return an error in response.
	if minioInstance, ok := api.objectMap[objectId]; ok {
		content, err := api.minioAccessor.getMinioInstanceObject(objectId, minioInstance)
		if err != nil {
			http.Error(w, "Error during fetching object from instance.", http.StatusInternalServerError)
			return
		}
		w.Write(content)
	} else {
		http.Error(w, fmt.Sprintf("Object with id %s not found on any of underlying instances.", objectId), http.StatusBadRequest)
	}
}

func (api *API) handleObjectPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	objectId := vars["id"]
	// if the objectId is already present in memory, return an error.
	if _, ok := api.objectMap[objectId]; ok {
		http.Error(w, fmt.Sprintf("Object with id %s is already saved to the storage, please consider using different id.", objectId), http.StatusBadRequest)
		return
	}

	// choose one instance at random and assign the objectId to map of ids: instances.
	instance := api.roundRobin()
	api.objectMap[objectId] = instance

	err := api.minioAccessor.sendContentToMinioInstance(objectId, instance, r.Body, int64(1024))
	if err != nil {
		http.Error(w, "Error during sending object to instance.", http.StatusInternalServerError)
	}

	w.Write([]byte("Object saved successfully."))
}

// roundRobin returns the least commonly used instance for next POST request.
// It sorts the instances by filesPushedThisRuntime and returns the one with lowest value.
func (api *API) roundRobin() MinioInstance {
	sort.Slice(api.minioInstances, func(i, j int) bool {
		return api.minioInstances[i].filesPushedThisRuntime < api.minioInstances[j].filesPushedThisRuntime
	})
  api.minioInstances[0].filesPushedThisRuntime++ // bump the counter
	return api.minioInstances[0]
}
