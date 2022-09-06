package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

type InfuraIpfsClient struct {
	ProjectID     string
	ProjectSecret string
	EndPoint      string
}

// Will return a client with this project id and secret for infura
func NewInfuraIpfsClient(projectID string, projectSecret string) (*InfuraIpfsClient, error) {
	new := &InfuraIpfsClient{
		ProjectID:     projectID,
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
