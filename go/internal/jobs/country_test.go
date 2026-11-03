package jobs

import "testing"

func TestParseCountryFromName(t *testing.T) {
	cases := map[string]string{
		"":               "",
		"🇭🇰 香港 01":       "HK",
		"🇺🇸 Los Angeles": "US",
		"日本 东京 IEPL":     "JP",
		"Singapore-02":   "SG",
		"RUSSIA node":    "RU",
		"Prussia relay":  "",
		"IN-mumbai":      "IN",
		"plain name":     "",
		"US-West":        "US",
		"BUSINESS":       "",
	}
	for in, want := range cases {
		if got := ParseCountryFromName(in); got != want {
			t.Errorf("ParseCountryFromName(%q) = %q, want %q", in, got, want)
		}
	}
}
