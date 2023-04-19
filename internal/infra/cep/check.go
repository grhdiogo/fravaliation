package cep

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type CepResponse struct {
	Bairro      string `json:"bairro"`
	Cep         string `json:"cep"`
	Cidade      string `json:"cidade"`
	Complemento string `json:"complemento"`
	End         string `json:"end"`
	Estado      string `json:"uf"`
	Unidade     string `json:"unidade"`
}

// checkZipCode validate if ZipCode is valid or not
func CheckZipCode(zipCode string) bool {
	zipCode = strings.ReplaceAll(zipCode, "-", "")
	resp, err := http.Get(fmt.Sprintf("https://viacep.com.br/ws/%s/json/", zipCode))
	if err != nil {
		fmt.Println("Erro ao consultar o CEP:", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Erro ao ler o corpo da resposta:", err)
		return false
	}
	//
	cepResp := new(CepResponse)
	//
	err = json.Unmarshal(body, cepResp)
	if err != nil {
		fmt.Println("Erro ao decodificar a resposta JSON:", err)
		return false
	}

	if cepResp.Cep == "" {
		return false
	}

	return true
}
