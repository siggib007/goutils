package utils

import "testing"

func TestIsInt(t *testing.T) {
	lstValid := []struct {
		strInput string
		strDesc  string
	}{
		{"42", "positive integer"},
		{"0", "zero"},
		{"1", "single digit"},
		{"999999", "large integer"},
	}
	for _, objTest := range lstValid {
		if !IsInt(objTest.strInput) {
			t.Errorf("expected '%s' (%s) to be valid int but got invalid", objTest.strInput, objTest.strDesc)
		}
	}

	lstInvalid := []struct {
		strInput string
		strDesc  string
	}{
		{"", "empty string"},
		{"abc", "letters"},
		{"42.5", "float"},
		{"-1", "negative"},
		{"12 34", "space in middle"},
		{"12.0", "float with zero decimal"},
	}
	for _, objTest := range lstInvalid {
		if IsInt(objTest.strInput) {
			t.Errorf("expected '%s' (%s) to be invalid int but got valid", objTest.strInput, objTest.strDesc)
		}
	}
}

func TestParseFloat(t *testing.T) {
	lstTests := []struct {
		strInput  string
		fExpected float64
		strDesc   string
	}{
		{"42.5", 42.5, "basic float"},
		{"0.0", 0.0, "zero"},
		{"100", 100.0, "whole number"},
		{"3.14159", 3.14159, "pi"},
		{"", 0.0, "empty string returns zero"},
		{"abc", 0.0, "invalid returns zero"},
	}
	for _, objTest := range lstTests {
		fResult := ParseFloat(objTest.strInput)
		if fResult != objTest.fExpected {
			t.Errorf("ParseFloat('%s') (%s): expected %f got %f",
				objTest.strInput, objTest.strDesc, objTest.fExpected, fResult)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	lstTests := []struct {
		strName    string
		strPattern string
		bExpected  bool
		strDesc    string
	}{
		{"12345_receipt.pdf", "12345*", true, "matches prefix"},
		{"99999_receipt.pdf", "12345*", false, "different prefix"},
		{"12345", "12345", true, "exact match"},
		{"12346", "12345", false, "no match no wildcard"},
		{"", "12345*", false, "empty name"},
	}
	for _, objTest := range lstTests {
		bResult := MatchPattern(objTest.strName, objTest.strPattern)
		if bResult != objTest.bExpected {
			t.Errorf("MatchPattern('%s', '%s') (%s): expected %v got %v",
				objTest.strName, objTest.strPattern, objTest.strDesc,
				objTest.bExpected, bResult)
		}
	}
}
