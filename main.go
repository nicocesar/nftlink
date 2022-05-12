package main

import (
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
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"

	"github.com/spf13/viper"

	_ "net/http/pprof"

	"github.com/philippgille/gokv/datastore"
	"github.com/philippgille/gokv/encoding"
)

type ClaimPrize struct {
	UUID    string `json:"uuid"`
	Claimed bool   `json:"claimed"`
	Wallet  string `json:"wallet"` // saving the wallet just in case the request to the blockchain fails, this dies process dies and we need to retry
}

// content holds our static web server content.
//go:embed web/build
var content embed.FS

// RandomString returns a random string of the given length.
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

var initFlag = flag.Bool("init", false, "initialize the database")

func main() {
	viper.SetConfigName("config")         // name of config file (without extension)
	viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/nftlink/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.nftlink") // call multiple times to add many search paths
	viper.AddConfigPath(".")              // optionally look for config in the working directory
	err := viper.ReadInConfig()           // Find and read the config file
	if err != nil {                       // Handle errors reading the config file
		fmt.Printf("fatal error config file: %s", err)
	}
	viper.SetEnvPrefix("nftlink") // will be uppercased automatically
	viper.BindEnv("contract_address")
	viper.BindEnv("private_key")
	viper.BindEnv("gas_limit")
	viper.BindEnv("gas_price")
	viper.BindEnv("infura_project_id")
	viper.BindEnv("infura_project_secret")
	viper.BindEnv("ethereum_client")

	flag.Parse()

	// Initialize the database or store.
	// this database wwill have the list of (reedemed) codes
	//options := file.DefaultOptions // change as necesary
	options := datastore.Options{
		ProjectID:       "qrcodenft",
		CredentialsFile: "",
		Codec:           encoding.JSON,
	}
	store, err := datastore.NewClient(options)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Populate the database with some random data if init flag is set
	if initFlag != nil && *initFlag {
		// Initialize the store
		for i := 0; i < 1000; i++ {
			key := RandomString(10)
			val := ClaimPrize{UUID: key, Claimed: false}
			err := store.Set(key, val)
			if err != nil {
				panic(err)
			}
		}
	}
	// Setup the IPFS client

	ipfs, err := NewInfuraIpfsClient(viper.GetString("infura_project_id"), viper.GetString("infura_project_secret"))
	if err != nil {
		panic(err)
	}
	r := mux.NewRouter()

	client, err := ethclient.Dial(viper.GetString("ethereum_client"))
	if err != nil {
		panic(err)
	}

	m := &minter{
		store:           store,
		ipfs:            ipfs,
		client:          client,
		privateKey:      viper.GetString("private_key"),
		contractAddress: viper.GetString("contract_address"),
		gasLimit:        uint64(viper.GetInt32("gas_limit")),
		gasPrice:        big.NewInt(viper.GetInt64("gas_price") * 1000000000),
	}
	r.Handle("/mint/{id}/{wallet}", m)

	checker := &checker{store: store}
	r.Handle("/check/{id}", checker)

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
	router.PathPrefix(endpoint).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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
