package model

type ECSInstance struct {
	ID          string
	Name        string
	Status      string
	Type        string
	Zone        string
	PrivateIP   string
	PublicIP    string
	OSName      string
	OSType      string
	ImageID     string
	ChargeType  string
	// ExpiredTime is Alibaba Cloud's ISO8601 renewal/expiry timestamp. It is
	// only meaningful for PrePaid (subscription) instances; PostPaid
	// (pay-as-you-go) instances don't expire and this is ignored for them.
	ExpiredTime string
	VPCID       string
	VPCName     string
	VPCCIDR     string
	VSwitchID   string
	VSwitchName string
	VSwitchCIDR string
	// ENIID is the primary Elastic Network Interface ID attached to the
	// instance's primary network card.
	ENIID      string
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

// Backward-compatible aliases for older internal packages.
type Bill = BillSummary
type RunResult = CommandResult
