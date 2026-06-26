package kennitala

import "testing"

func TestValidateKT(t *testing.T) {
	lstValid := []string{
		"0101103190",
		"010110-3190",
		"010110 3190",
	}
	for _, strKT := range lstValid {
		if !ValidateKT(strKT) {
			t.Errorf("expected %s to be valid but got invalid", strKT)
		}
	}

	lstInvalid := []string{
		"",
		"1234",
		"abcdefghij",
		"0101103180",
		"0101103170",
		"9901103190",
		"0101103100",
	}
	for _, strKT := range lstInvalid {
		if ValidateKT(strKT) {
			t.Errorf("expected %s to be invalid but got valid", strKT)
		}
	}
}
