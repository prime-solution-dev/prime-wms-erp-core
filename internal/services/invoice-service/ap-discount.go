package invoiceService

func calculateAPDiscount(totalBeforeDiscount, percent, amount float64) float64 {
	if percent > 0 {
		return round2(totalBeforeDiscount * percent / 100)
	}
	return round2(amount)
}
