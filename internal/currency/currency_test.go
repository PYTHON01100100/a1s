package currency

import "testing"

func TestUSDToSAR(t *testing.T) {
	c := Converter{SARPerUSD: 3.75}
	got := c.Convert(100, "USD", "SAR")
	if got != 375 {
		t.Fatalf("want 375, got %v", got)
	}
}

func TestSARToUSD(t *testing.T) {
	c := Converter{SARPerUSD: 3.75}
	got := c.Convert(375, "SAR", "USD")
	if got != 100 {
		t.Fatalf("want 100, got %v", got)
	}
}
