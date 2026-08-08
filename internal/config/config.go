package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Region    string
	Profile   string
	Currency  string
	SARPerUSD float64
	ReadOnly  bool
	Demo      bool
	AIBaseURL string
	AIAPIKey  string
	AIModel   string
}

func FromEnv() Config {
	fx := 3.75
	if v := os.Getenv("A1S_SAR_PER_USD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			fx = f
		}
	}
	currency := strings.ToUpper(os.Getenv("A1S_CURRENCY"))
	if currency == "" {
		currency = "USD"
	}
	return Config{
		Region:    os.Getenv("ALIBABA_CLOUD_REGION_ID"),
		Profile:   os.Getenv("ALIBABA_CLOUD_PROFILE"),
		Currency:  currency,
		SARPerUSD: fx,
		AIBaseURL: os.Getenv("A1S_AI_BASE_URL"),
		AIAPIKey:  os.Getenv("A1S_AI_API_KEY"),
		AIModel:   os.Getenv("A1S_AI_MODEL"),
	}
}
