package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	nftlink "github.com/nicocesar/nftlink/lib/contracts/nftlink"
	"github.com/philippgille/gokv"
	"github.com/spf13/viper"
)

type worker struct {
	store gokv.Store
	ipfs  IIPFSClient
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
