package currency

import "strings"

type Converter struct{ SARPerUSD float64 }

func (c Converter) Convert(amount float64, from, to string) float64 {
	from, to = strings.ToUpper(from), strings.ToUpper(to)
	if from == to || amount == 0 {
		return amount
	}
	if c.SARPerUSD <= 0 {
		c.SARPerUSD = 3.75
	}
	if from == "USD" && to == "SAR" {
		return amount * c.SARPerUSD
	}
	if from == "SAR" && to == "USD" {
		return amount / c.SARPerUSD
	}
	return amount
}

func Symbol(code string) string {
	switch strings.ToUpper(code) {
	case "SAR":
		return "SAR"
	case "USD":
		return "$"
	default:
		return strings.ToUpper(code)
	}
}
