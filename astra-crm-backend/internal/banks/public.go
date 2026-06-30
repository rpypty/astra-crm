package banks

type PublicBank struct {
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	CSVAlias *string `json:"csvAlias,omitempty"`
}

func PublicBankFromDomain(bank Bank) PublicBank {
	return PublicBank{
		Code:     bank.Code,
		Name:     bank.Name,
		CSVAlias: bank.CSVAlias,
	}
}

func PublicBanks(items []Bank) []PublicBank {
	result := make([]PublicBank, 0, len(items))
	for _, item := range items {
		result = append(result, PublicBankFromDomain(item))
	}

	return result
}
