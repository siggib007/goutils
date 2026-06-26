package kennitala

import "strings"

// ValidateKT takes an Icelandic kennitala and validates it.
// Accepts kennitala with or without separators (-, space, ., ,)
// Returns true if valid, false otherwise.
func ValidateKT(strKennitala string) bool {
	strKennitala = strings.NewReplacer("-", "", " ", "", ".", "", ",", "").Replace(strKennitala)

	if len(strKennitala) != 10 {
		return false
	}
	for _, c := range strKennitala {
		if c < '0' || c > '9' {
			return false
		}
	}

	lstCheck := []int{3, 2, 7, 6, 5, 4, 3, 2}
	iSum := 0
	for iIndex, iWeight := range lstCheck {
		iDigit := int(strKennitala[iIndex] - '0')
		iSum += iDigit * iWeight
	}

	iCheck := iSum % 11
	iExpected := 11 - iCheck
	return iExpected == int(strKennitala[8]-'0')
}
