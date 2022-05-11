package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	nftlink "github.com/nicocesar/nftlink/lib/contracts/nftlink"
	"github.com/philippgille/gokv"
)

type ethBackend interface {
	bind.ContractBackend
	bind.DeployBackend
	ethereum.TransactionReader
	NetworkID(ctx context.Context) (*big.Int, error)
}

type minter struct {
	store           gokv.Store
	ipfs            IIPFSClient
	client          ethBackend // this might have to be an interface for testing
	privateKey      string
	contractAddress string
	gasLimit        uint64
	gasPrice        *big.Int
}

func (m *minter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]
	wallet := vars["wallet"]

	retrievedVal := &ClaimPrize{}
	found, err := m.store.Get(key, &retrievedVal)
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

	if m.client == nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "You must set the ethclient")
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	nftAddress := common.HexToAddress(m.contractAddress)
	nftcontract, err := nftlink.NewNFTLink(nftAddress, m.client)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	privateKey, err := crypto.HexToECDSA(m.privateKey)
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

	nonce, err := m.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	chainID, err := m.client.NetworkID(context.Background())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

	// TODO: get this from config
	value := big.NewInt(0) // in wei (0 eth)
	gasLimit := m.gasLimit // in units
	gasPrice := m.gasPrice // in wei

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

	cid, err := m.ipfs.Add(strings.NewReader(string(metadataJson)))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

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
	err = m.store.Set(key, val)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}

}
