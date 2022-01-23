package main

import (
	"context"
	"crypto/ecdsa"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	_ "net/http/pprof"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/file"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	nftlink "github.com/nicocesar/nftlink/lib/contracts/nftlink"
)

type ClaimPrize struct {
	UUID    string `json:"uuid"`
	Claimed bool   `json:"claimed"`
	Wallet  string `json:"wallet"` // saving the wallet just in case the request to the blockchain fails, this dies process dies and we need to retry
}

// content holds our static web server content.
//go:embed web/build
var content embed.FS

type worker struct {
	store gokv.Store
}

func (worker *worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]
	wallet := vars["wallet"]

	retrievedVal := &ClaimPrize{}
	found, err := worker.store.Get(key, &retrievedVal)
	if err != nil {
		panic(err)
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Key %s not found", key)
		return
	}

	A, err := common.NewMixedcaseAddressFromString(wallet)

	if err != nil || !A.ValidChecksum() {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid wallet address")
		return
	}

	if retrievedVal.Claimed {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Already claimed")
		return
	}

	client, err := ethclient.Dial("http://localhost:8545")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
	// TODO:let's mint a NFT and give it to the wallet
	//
	nftAddress := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	nftcontract, err := nftlink.NewNFTLink(nftAddress, client)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	//Now we can read the nonce that we should use for the account's transaction.

	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	// TODO: get this from config
	value := big.NewInt(0)              // in wei (0 eth)
	gasLimit := uint64(30000)           // in units
	gasPrice := big.NewInt(30000000000) // in wei (30 gwei)

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	opts := &bind.TransactOpts{
		Nonce:    new(big.Int).SetUint64(nonce),
		From:     fromAddress,
		Signer:   auth.Signer,
		Value:    value,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
	}
	// FIXME! Safe mint should point to the IPFS metadata of the NFT
	rtn_tx, err := nftcontract.NFTLinkTransactor.SafeMint(opts, A.Address(), "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	b, err := rtn_tx.MarshalJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", b)

	// prize has been claimed now let's write it to the database
	val := ClaimPrize{UUID: key, Wallet: wallet, Claimed: true}
	err = worker.store.Set(key, val)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

}

// RandomString returns a random string of the given length.
// this is *deterministic* and safe for use in tests and dev.
// in production, replace this for a proper random function
// maybe from the database?
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

var initFlag = flag.Bool("init", false, "initialize the database")

func main() {
	flag.Parse()
	options := file.DefaultOptions //

	// Create client
	store, err := file.NewStore(options)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	if initFlag != nil && *initFlag {
		// Initialize the store
		for i := 0; i < 10; i++ {
			key := RandomString(10)
			val := ClaimPrize{UUID: key, Claimed: false}
			err := store.Set(key, val)
			if err != nil {
				panic(err)
			}
		}
	}

	r := mux.NewRouter()

	minter := &worker{store: store}
	r.Handle("/mint/{id}/{wallet}", minter)

	// Add some profiling.
	r.Handle("/debug/pprof/profile", http.DefaultServeMux)
	r.Handle("/debug/pprof/heap", http.DefaultServeMux)

	fileServer(r, "/", content, "web/build")

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
	}
	// Start HTTP server.
	log.Printf("listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// FileServer is serving static files.
func fileServer(router *mux.Router, endpoint string, rootFS fs.FS, root string) {
	// Strip the "web/build" prefix, or whatever root is
	strippedFS, err := fs.Sub(rootFS, root)
	if err != nil {
		log.Fatal(err)
	}

	relocatedFS := http.FS(strippedFS)
	staticHandler := http.FileServer(relocatedFS)

	router.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {

		if _, err := relocatedFS.Open(r.RequestURI); err != nil {
			// the file is not there...
			uriWithoutQuery := strings.Split(r.RequestURI, "?")[0]
			v := http.StripPrefix(uriWithoutQuery, staticHandler)
			v.ServeHTTP(w, r)
		} else {
			// the file is there (most likely static content), just server it...
			staticHandler.ServeHTTP(w, r)
		}
	})
}
