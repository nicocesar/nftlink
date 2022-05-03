package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/philippgille/gokv/syncmap"
)

func TestChecker(t *testing.T) {
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
		{"Unredeemed code", "U6fxRAqxMo", false, "", "GET", "/check/U6fxRAqxMo", http.StatusOK, `Redeem code U6fxRAqxMo found`},
		{"Redeemed code", "U6fxRAqxMo", true, "", "GET", "/check/U6fxRAqxMo", http.StatusOK, `Redeem code U6fxRAqxMo found`},
		{"Unreedemed code not found", "", false, "", "GET", "/check/notfound12", http.StatusNotFound, `Redeem code notfound12 not found`},
		{"Reedemed code not found", "", true, "", "GET", "/check/notfound12", http.StatusNotFound, `Redeem code notfound12 not found`},
		{"POST method also valid", "U6fxRAqxMo", false, "", "POST", "/check/U6fxRAqxMo", http.StatusOK, `Redeem code U6fxRAqxMo found`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an inmemory store for testing
			options := syncmap.Options{}
			store := syncmap.NewStore(options)
			defer store.Close()

			worker := checker{
				store: store,
			}

			worker.store.Set(tc.uuid, &ClaimPrize{
				UUID:    tc.uuid,
				Claimed: tc.claimed,
				Wallet:  tc.wallet,
			})

			// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()

			// create an HTTP request for the checker
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			// we need a router to process URL parameters and other things
			r := mux.NewRouter()
			r.Handle("/check/{id}", &worker)

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
