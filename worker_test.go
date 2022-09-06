package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/nicocesar/nftlink/lib/contracts/nftlink"
	"github.com/philippgille/gokv/syncmap"
)

type ipfsMock struct {
}

func (i ipfsMock) Add(input io.Reader) (IPFSUploadResponse, error) {
	return IPFSUploadResponse{}, nil
}

/* helper function that could be used in the future
// Returns a channel that blocks until the transaction is confirmed
func waitTxConfirmed(ctx context.Context, c ethBackend, hash common.Hash) <-chan *types.Transaction {
	ch := make(chan *types.Transaction)
	go func() {
		for {
			tx, pending, _ := c.TransactionByHash(ctx, hash)
			if !pending {
				ch <- tx
			}

			time.Sleep(time.Millisecond * 500)
		}
	}()

	return ch
}
*/

func TestWorker(t *testing.T) {
	var cases = []struct {
		name               string
		uuid               string
		claimed            bool
		wallet             string
		method             string
		url                string
		expectedStatus     int
		expectedBodyRegexp string
	}{
		{"Unredeemed code", "U6fxRAqxMo", false, "0x", "GET", "/mint/U6fxRAqxMo/0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", http.StatusOK, `"status":"success"`},
		{"Unredeemed code invalid wallet", "U6fxRAqxMo", false, "0x", "GET", "/mint/U6fxRAqxMo/0x123456", http.StatusBadRequest, `Invalid wallet address`},
		{"Already redeemed", "U6fxRAqxMo", true, "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", "GET", "/mint/U6fxRAqxMo/0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", http.StatusOK, `Already claimed`},
		{"Redeem code not found", "U6fxRAqxMo", false, "0x", "GET", "/mint/not_found/0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B", http.StatusNotFound, `Redeem code not_found not found`},
	}

	deployerKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Create an inmemory store for testing
			options := syncmap.Options{}
			store := syncmap.NewStore(options)
			defer store.Close()

			ipfsMock := &ipfsMock{}
			clientMock := NewSimulatedBackend()

			deployerPublicKey := deployerKey.Public()
			deployerPublicKeyECDSA, ok := deployerPublicKey.(*ecdsa.PublicKey)
			if !ok {
				t.Errorf("error casting public key to ECDSA")
			}

			deployerFromAddress := crypto.PubkeyToAddress(*deployerPublicKeyECDSA)
			nonce, err := clientMock.PendingNonceAt(context.Background(), deployerFromAddress)
			if err != nil {
				t.Fatal(err)
			}
			gasPrice, err := clientMock.SuggestGasPrice(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			networkID, err := clientMock.NetworkID(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			auth, err := bind.NewKeyedTransactorWithChainID(deployerKey, networkID)
			if err != nil {
				t.Fatal(err)
			}

			auth.Nonce = big.NewInt(int64(nonce))
			auth.Value = big.NewInt(0)      // in wei
			auth.GasLimit = uint64(3000000) // in units
			auth.GasPrice = gasPrice
			auth.From = deployerFromAddress

			clientMock.FundAddress(context.Background(), deployerFromAddress)

			address, _, _, err := nftlink.DeployNFTLink(auth, clientMock)
			if err != nil {
				t.Fatal(err)
			}
			/*
				c := waitTxConfirmed(context.Background(), clientMock, tx.Hash())

				// wait for the transaction to be confirmed
				tx = <-c

				fmt.Printf("Deployed contract at %s, transaction to: %s \n", address.Hex(), tx.To())
			*/
			m := minter{
				store:           store,
				ipfs:            ipfsMock,
				client:          clientMock,
				privateKey:      fmt.Sprintf("%x", crypto.FromECDSA(deployerKey)),
				contractAddress: address.Hex(),
				gasLimit:        3000000,
				gasPrice:        gasPrice, //big.NewInt(1000000000),
			}

			m.store.Set(tc.uuid, &ClaimPrize{
				UUID:    tc.uuid,
				Claimed: tc.claimed,
				Wallet:  tc.wallet,
			})

			// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
			rr := httptest.NewRecorder()

			// create an HTTP request for the minter
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			// we need a router to process URL parameters and other things
			r := mux.NewRouter()
			r.Handle("/mint/{id}/{wallet}", &m)

			// serve the request
			r.ServeHTTP(rr, req)

			// Check the status code is what we expect.
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got '%v' want '%v'",
					status, tc.expectedStatus)
			}

			// Check the response body is what we expect.
			match, err := regexp.MatchString(tc.expectedBodyRegexp, rr.Body.String())
			if err != nil || !match {
				t.Errorf("handler can't match regexp '%v' in body '%v'",
					tc.expectedBodyRegexp, rr.Body.String())
			}
		})
	}
}
