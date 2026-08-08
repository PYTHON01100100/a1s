package model

type ECSInstance struct {
	ID         string
	Name       string
	Status     string
	Type       string
	Zone       string
	PrivateIP  string
	PublicIP   string
	CPUPercent float64
}

type MetricPoint struct {
	Timestamp int64
	Average   float64
	Maximum   float64
	Minimum   float64
}

type CommandResult struct {
	InvokeID string
	Status   string
	Output   string
	ExitCode int
}

type BillSummary struct {
	BillingCycle string
	PretaxAmount float64
	Currency     string
}

type PriceQuote struct {
	InstanceType  string
	PriceUnit     string
	OriginalPrice float64
	DiscountPrice float64
	TradePrice    float64
	Currency      string
}
