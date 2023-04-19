package appl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fravaliation/internal/domain/quote"
	"fravaliation/internal/infra/cep"
	"fravaliation/internal/infra/db/postgres"
	"fravaliation/internal/infra/fr"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// TODO: Remover daqui?
const (
	CNPJ         = "25438296000158"
	AuthToken    = "1d52a9b6b78cf07b08586152459a5c90"
	PlatformCode = "5AKVkHqCn"
	BaseUrl      = "https://sp.freterapido.com"
)

type QuoteService interface {
	CreateQuoteFreight(p CreateQuoteParams) (*CreateQuoteResponse, error)
	Metricts(limit int) (*Metric, error)
}

type quoteServiceImpl struct {
	ctx context.Context
}

type CreateQuoteVolumeParams struct {
	Category      int
	Amount        int
	UnitaryWeight float64
	Price         float64
	Sku           string
	Height        float64
	Width         float64
	Length        float64
}

type CreateQuoteParams struct {
	AddressZipCode string
	Volumes        []CreateQuoteVolumeParams
}

type Carrier struct {
	Name     string
	Service  string
	Deadline string
	Price    float64
}

type CreateQuoteResponse struct {
	Carriers []Carrier
}

type CarrierMetrict struct {
	ResultQuantity       int
	TotalValue           float64
	MostExpensiveFreight float64
	CheaperFreight       float64
}

type Metric struct {
	CarriersMetrics map[string]CarrierMetrict
}

func (s *quoteServiceImpl) validate(p CreateQuoteParams) error {
	var errs = make([]string, 0)
	_, cepErr := strconv.Atoi(p.AddressZipCode)
	// validate cep
	if cepErr != nil {
		errs = append(errs, "CEP inválido, digite apenas números")
	}
	if !cep.CheckZipCode(p.AddressZipCode) || cepErr != nil {
		errs = append(errs, "CEP não existe")
	}
	if len(p.Volumes) == 0 {
		errs = append(errs, "Ao menos 1(um) volume deve ser passado")
	}
	for index, v := range p.Volumes {
		i := index + 1
		// Category
		if fr.CategoryMapping[v.Category] == "" {
			errs = append(errs, fmt.Sprintf("Categoria do %dº volume é inválido", i))
		}
		// Amount
		if v.Amount <= 0 {
			errs = append(errs, fmt.Sprintf("Quantidade do %dº volume é inválido", i))
		}
		// UnitaryWeight
		if v.UnitaryWeight <= 0 {
			errs = append(errs, fmt.Sprintf("Peso unitário do %dº volume é inválido", i))
		}
		// Price
		if v.Price <= 0 {
			errs = append(errs, fmt.Sprintf("Preço do %dº volume é inválido", i))
		}
		// Sku
		if len(v.Sku) > 255 {
			errs = append(errs, fmt.Sprintf("Quantidade de caracteres de Sku do %dº volume é muito grande", i))
		}
		// Height
		if v.Height <= 0 {
			errs = append(errs, fmt.Sprintf("Altura do %dº volume é inválido", i))
		}
		// Width
		if v.Width <= 0 {
			errs = append(errs, fmt.Sprintf("Largura do %dº volume é inválido", i))
		}
		// Length
		if v.Length <= 0 {
			errs = append(errs, fmt.Sprintf("Tamanho do %dº volume é inválido", i))
		}
	}
	// case exist error
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	// success
	return nil
}

func (s *quoteServiceImpl) CreateQuoteFreight(p CreateQuoteParams) (*CreateQuoteResponse, error) {
	// validate params
	err := s.validate(p)
	if err != nil {
		return nil, err
	}
	// new client
	cli := fr.NewFrClient(fr.Config{
		BaseUrl: BaseUrl,
	})
	// validation already made on validate func
	zipCode, _ := strconv.Atoi(p.AddressZipCode)
	// volumes
	volumes := make([]fr.DispatcherVolume, 0)
	vlms := make([]quote.Volume, 0)
	// create volumes request struct and volumes for quote entity
	for _, v := range p.Volumes {
		// request volume struct
		volumes = append(volumes, fr.DispatcherVolume{
			Sku:           v.Sku,
			Amount:        v.Amount,                 //        int     `json:"amount"`         // required
			Category:      strconv.Itoa(v.Category), //      string  `json:"category"`       // required
			Height:        v.Height,                 //        float64 `json:"height"`         // required
			Width:         v.Width,                  //         float64 `json:"width"`          // required
			Length:        v.Length,                 //        float64 `json:"length"`         // required
			UnitaryPrice:  v.Price,                  //  float64 `json:"unitary_price"`  // required
			UnitaryWeight: v.UnitaryWeight,          // float64 `json:"unitary_weight"` // required
		})
		// quote volume struct
		vlms = append(vlms, quote.Volume{
			Category:      v.Category,
			Amount:        v.Amount,
			UnitaryWeight: v.UnitaryWeight,
			Price:         v.Price,
			Sku:           v.Sku,
			Height:        v.Height,
			Width:         v.Width,
			Length:        v.Length,
		})
	}
	// create request
	req := &fr.CreateFreightQuotationRequest{
		Shipper: fr.Shipper{
			RegisteredNumber: CNPJ,
			Token:            AuthToken,
			PlatformCode:     PlatformCode,
		},
		Recipient: fr.Recipient{
			Type:    fr.RecipientNaturalPerson,
			Country: "BRA",
			// TODO: CEP?
			Zipcode: zipCode,
		},
		Dispatchers: []fr.Dispatcher{
			{
				RegisteredNumber: CNPJ,
				Volumes:          volumes,
				// TODO: CEP?
				Zipcode: zipCode,
			},
		},
		// TODO: Qual dos?
		SimulationType: []fr.ReturnSimulationTypeKind{
			fr.ReturnSimulationTypeFract,
		},
	}
	// make quotation
	response, err := cli.CreateFreight(req)
	if err != nil {
		return nil, errors.New("Falha ao recuperar cotação de frete")
	}
	// get conn
	tx, err := postgres.GetInstance().GetConn()
	if err != nil {
		return nil, errors.New("Falhar ao conectar com banco de dados")
	}
	//
	rep := postgres.NewQuoteRepository(s.ctx, tx)
	rawResponse, err := json.Marshal(response)
	if err != nil {
		return nil, errors.New("Falha ao transformar dados de resposta da cotação")
	}
	rawReq, _ := json.Marshal(req)
	// store
	err = rep.Store(quote.Entity{
		ID:      uuid.New().String(),
		CpfCnpj: CNPJ,
		Address: quote.Address{
			Cep: p.AddressZipCode,
		},
		RawResponse: rawResponse,
		RawRequest:  rawReq,
		Volumes:     vlms,
	})
	if err != nil {
		return nil, errors.New("Falha ao salvar cotação")
	}
	// response
	carriers := make([]Carrier, 0)
	for _, v := range response.Dispatchers {
		for _, v1 := range v.Offers {
			carriers = append(carriers, Carrier{
				Name: v1.Carrier.Name,
				// TODO: Que campo é esse
				Service: v1.Service,
				// TODO: Que campo é esse
				Deadline: v1.DeliveryTime.EstimatedDate,
				Price:    v1.FinalPrice,
			})
		}
	}
	// commit transaction
	err = tx.Commit(s.ctx)
	if err != nil {
		return nil, errors.New("Falha ao salvar dados")
	}
	// success
	return &CreateQuoteResponse{
		Carriers: carriers,
	}, nil
}

func (s *quoteServiceImpl) Metricts(limit int) (*Metric, error) {
	// get conn
	tx, err := postgres.GetInstance().GetConn()
	if err != nil {
		return nil, errors.New("Falhar ao conectar com banco de dados")
	}
	// list quotations
	rep := postgres.NewQuoteRepository(s.ctx, tx)

	list, err := rep.List(limit)
	if err != nil {
		return nil, errors.New("Falha ao listar cotações")
	}
	// result
	result := &Metric{
		CarriersMetrics: map[string]CarrierMetrict{},
	}
	// decode responses
	for _, v := range list {
		resp := new(fr.CreateFreightQuotationResponse)
		// decode response
		err = json.Unmarshal(v.RawResponse, resp)
		if err != nil {
			return nil, errors.New("Falha ao decodificar cotação")
		}
		// create metrics
		for _, dispatcher := range resp.Dispatchers {
			for _, offer := range dispatcher.Offers {
				//
				old := result.CarriersMetrics[offer.Carrier.Name]
				// add first value
				if old.ResultQuantity == 0 {
					old.CheaperFreight = offer.FinalPrice
				}
				//
				new := CarrierMetrict{
					ResultQuantity:       old.ResultQuantity + 1,
					TotalValue:           old.TotalValue + offer.FinalPrice,
					MostExpensiveFreight: bigger(old.MostExpensiveFreight, offer.FinalPrice),
					CheaperFreight:       smaller(old.CheaperFreight, offer.FinalPrice),
				}
				// replace
				result.CarriersMetrics[offer.Carrier.Name] = new
			}
		}

	}

	return result, nil
}

func NewQuoteService(ctx context.Context) QuoteService {
	return &quoteServiceImpl{
		ctx: ctx,
	}
}
