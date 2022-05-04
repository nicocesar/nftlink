package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/philippgille/gokv/syncmap"
)

type ipfsMock struct {
}

func (i ipfsMock) Add(input io.Reader) (IPFSUploadResponse, error) {
	return IPFSUploadResponse{}, nil
}

func TestWorker(t *testing.T) {
	var cases = []struct {
		name           string
		uuid           string
		claimed        bool
		wallet         string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		// FIXME: mocking ethclient is needed
		// see https://medium.com/@m.vanderwijden1/intro-to-web3-go-part-4-5a21bc71fddc
		// on how to implement a eth.Client mock
		// {"Unredeemed code", "U6fxRAqxMo", false, "0x", "GET", "/mint/U6fxRAqxMo/0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", http.StatusOK, `Redeem code U6fxRAqxMo found`},
		{"Unredeemed code invalid wallet", "U6fxRAqxMo", false, "0x", "GET", "/mint/U6fxRAqxMo/0x123456", http.StatusBadRequest, `Invalid wallet address`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an inmemory store for testing
			options := syncmap.Options{}
			store := syncmap.NewStore(options)
			defer store.Close()

			ipfsMock := &ipfsMock{}

			worker := worker{
				store: store,
				ipfs:  ipfsMock,
			}

			worker.store.Set(tc.uuid, &ClaimPrize{
				UUID:    tc.uuid,
				Claimed: tc.claimed,
				Wallet:  tc.wallet,
			})

			// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()

			// create an HTTP request for the worker
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			// we need a router to process URL parameters and other things
			r := mux.NewRouter()
			r.Handle("/mint/{id}/{wallet}", &worker)

			// serve the request
			r.ServeHTTP(rr, req)

			// Check the status code is what we expect.
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// Check the response body is what we expect.
			if rr.Body.String() != tc.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tc.expectedBody)
			}
		})
	}
}
