package wa

import (
	"fravaliation/internal/interf"
	"fravaliation/internal/interf/wa/resource"
	"net/http"
)

var routes = []interf.RouteConfig{
	{
		Method:  http.MethodPost,
		Path:    "/quote",
		Handler: resource.CreateQuote,
	},
	{
		Method:  http.MethodGet,
		Path:    "/metrics",
		Handler: resource.Metrics,
	},
}
