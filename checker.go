package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/philippgille/gokv"
)

type checker struct {
	store gokv.Store
}

func (worker *checker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	retrievedVal := &ClaimPrize{}
	found, err := worker.store.Get(key, &retrievedVal)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Redeem code %s not found", key)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Redeem code %s found", key)
}
