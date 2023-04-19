package resource

import (
	"encoding/json"
	"fravaliation/internal/appl"
	"fravaliation/internal/interf"
	"net/http"
	"strconv"
)

// ========================================================================
// CREATE QUOTATION
// ========================================================================

type createQuoteVolumeRequest struct {
	Category      int     `json:"category"`
	Amount        int     `json:"amount"`
	UnitaryWeight float64 `json:"unitary_weight"`
	Price         float64 `json:"price"`
	Sku           string  `json:"sku"`
	Height        float64 `json:"height"`
	Width         float64 `json:"width"`
	Length        float64 `json:"length"`
}

type createQuoteRequest struct {
	Recipient struct {
		Address struct {
			Zipcode string `json:"zipcode"`
		} `json:"address"`
	} `json:"recipient"`
	Volumes []createQuoteVolumeRequest `json:"volumes"`
}

type createQuoteCarrierResponse struct {
	Name     string  `json:"name"`
	Service  string  `json:"service"`
	Deadline string  `json:"deadline"`
	Price    float64 `json:"price"`
}

type createQuoteResponse struct {
	Carrier []createQuoteCarrierResponse `json:"carrier"`
}

func CreateQuote(r *http.Request) (any, *interf.ErrorHandler) {
	request := new(createQuoteRequest)
	// decode body
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(request)
	if err != nil {
		return nil, &interf.ErrorHandler{
			StatusCode: http.StatusBadRequest,
			Err:        err,
			ErrCode:    999999999,
		}
	}
	// service
	app := appl.NewQuoteService(r.Context())
	// volumes
	vlms := make([]appl.CreateQuoteVolumeParams, 0)
	for _, v := range request.Volumes {
		vlms = append(vlms, appl.CreateQuoteVolumeParams{
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
	// make quotation
	resp, err := app.CreateQuoteFreight(appl.CreateQuoteParams{
		AddressZipCode: request.Recipient.Address.Zipcode,
		Volumes:        vlms,
	})
	if err != nil {
		return nil, &interf.ErrorHandler{
			StatusCode: http.StatusBadRequest,
			Err:        err,
			ErrCode:    1001,
		}
	}
	carriers := make([]createQuoteCarrierResponse, 0)
	for _, v := range resp.Carriers {
		carriers = append(carriers, createQuoteCarrierResponse{
			Name:     v.Name,
			Service:  v.Service,
			Deadline: v.Deadline,
			Price:    v.Price,
		})
	}

	// success
	return &createQuoteResponse{
		Carrier: carriers,
	}, nil
}

// ========================================================================
// METRICS
// ========================================================================

type metricsRequest struct {
	LastQuotes int
}

type carrierMetric struct {
	Name                 string  `json:"name"`
	ResultQuantity       int     `json:"resultQuantity"`
	TotalValue           float64 `json:"totalValue"`
	MostExpensiveFreight float64 `json:"mostExpensiveFreight"`
	CheaperFreight       float64 `json:"cheaperFreight"`
}

type metricsResponse struct {
	CarriersMetric []carrierMetric `json:"carriersMetric"`
}

func Metrics(r *http.Request) (any, *interf.ErrorHandler) {
	lq := r.URL.Query().Get("last_quotes")
	//
	lastQuotes, err := strconv.Atoi(lq)
	if err != nil || lastQuotes < 0 {
		// default limit is -1 to list all
		lastQuotes = -1
	}
	//
	request := &metricsRequest{
		LastQuotes: lastQuotes,
	}
	// service
	app := appl.NewQuoteService(r.Context())
	metric, err := app.Metricts(request.LastQuotes)
	if err != nil {
		return nil, &interf.ErrorHandler{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
			ErrCode:    1001,
		}
	}
	// success
	result := &metricsResponse{
		CarriersMetric: make([]carrierMetric, 0),
	}
	// adapt metrics to response
	for n, v := range metric.CarriersMetrics {
		result.CarriersMetric = append(result.CarriersMetric, carrierMetric{
			Name:                 n,
			ResultQuantity:       v.ResultQuantity,
			TotalValue:           v.TotalValue,
			MostExpensiveFreight: v.MostExpensiveFreight,
			CheaperFreight:       v.CheaperFreight,
		})
	}

	return result, nil
}
