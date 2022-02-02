package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"github.com/spf13/viper"

	_ "net/http/pprof"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/datastore"
	"github.com/philippgille/gokv/encoding"

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
	ipfs  IIPFSClient
}

type InfuraIpfsClient struct {
	ProjectID     string
	ProjectSecret string
	EndPoint      string
}

// Will return a client with this project id and secret for infura
func NewInfuraIpfsClient(projectID string, projectSecret string) (*InfuraIpfsClient, error) {
	new := &InfuraIpfsClient{
		ProjectID:     viper.GetString("datastore_project_id"),
		ProjectSecret: projectSecret,
		EndPoint:      "https://ipfs.infura.io:5001",
	}
	return new, nil
}

type IPFSUploadResponse struct {
	Hash string `json:"Hash"`
	Name string `json:"Name"`
	Size int64  `json:"Size"`
}

func (c *InfuraIpfsClient) Add(input io.Reader) (IPFSUploadResponse, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	defer mp.Close()

	part1, err := mp.CreateFormFile("file", "file")
	io.Copy(part1, input)
	if err != nil {
		return IPFSUploadResponse{}, err
	}

	request, err := http.NewRequest("POST", c.EndPoint+"/api/v0/add", body)
	request.Header.Add("Content-Type", mp.FormDataContentType())

	request.SetBasicAuth(c.ProjectID, c.ProjectSecret)
	if err != nil {
		return IPFSUploadResponse{}, err
	}

	//request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}

	response, err := client.Do(request)

	if err != nil {
		return IPFSUploadResponse{}, err
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return IPFSUploadResponse{}, err
	}

	if response.StatusCode != http.StatusOK {
		return IPFSUploadResponse{}, fmt.Errorf("%s", content)
	}

	// maybe a different name will be better?
	IPFSUploadResponse := IPFSUploadResponse{}
	json.Unmarshal(content, &IPFSUploadResponse)
	if err != nil {
		return IPFSUploadResponse, fmt.Errorf("error unmarshalling '%s': %e", content, err)
	}
	return IPFSUploadResponse, nil
}

type IIPFSClient interface {
	Add(input io.Reader) (IPFSUploadResponse, error)
}

func (worker *worker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]
	wallet := vars["wallet"]

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

	client, err := ethclient.Dial(viper.GetString("ethereum_client"))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	nftAddress := common.HexToAddress(viper.GetString("contract_address"))
	nftcontract, err := nftlink.NewNFTLink(nftAddress, client)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	privateKey, err := crypto.HexToECDSA(viper.GetString("private_key"))
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
	value := big.NewInt(0)                                           // in wei (0 eth)
	gasLimit := uint64(viper.GetInt32("gas_limit"))                  // in units
	gasPrice := big.NewInt(viper.GetInt64("gas_price") * 1000000000) // in wei (30 gwei)

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
	type Attribute struct {
		TraitType   string `json:"trait_type"`
		DisplayType string `json:"display_type,omitempty"`
		Value       string `json:"value"`
	}

	type Metadata struct {
		Name        string      `json:"name"`
		Description string      `json:"description"`
		Image       string      `json:"image"`
		Attributes  []Attribute `json:"attributes"`
	}

	number, err := nftcontract.NFTLinkCaller.Count(nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	metadata := Metadata{
		Name:        "Ma'hai #" + number.String(),
		Description: "Ma'hai #" + number.String(),
		Image:       "ipfs://QmWCsTr7EiVFpDsWkogrm7qidu2t7jHkiYVueCWCwD7ZA5",
		//Image: "ipfs://QmcxSwJ5PhYSArq1s1RS6Dt7Ji1m8YrDRQKPdt6Apizosw/mahai.jpeg",
		Attributes: []Attribute{
			{
				TraitType: "Producto",
				Value:     "London Dry Gin",
			},
			{
				TraitType: "Lote",
				Value:     "202112R",
			},
			{
				TraitType: "Partida",
				Value:     "840 botellas",
			},
			{
				TraitType: "Fabricado en",
				Value:     "Coronel Vidal, Buenos Aires, Argentina",
			},
			{
				TraitType:   "Fecha de produccion",
				DisplayType: "date",
				Value:       "1638421200",
			},
		},
	}

	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	cid, err := worker.ipfs.Add(strings.NewReader(string(metadataJson)))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	log.Printf("metadata hash: %s", cid.Hash)

	rtn_tx, err := nftcontract.NFTLinkTransactor.SafeMint(opts, A.Address(), cid.Hash)
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
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	 rand.Seed(time.Now().UnixNano())
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

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

	minter := &worker{store: store, ipfs: ipfs}
	r.Handle("/mint/{id}/{wallet}", minter)

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
