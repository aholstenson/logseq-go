package utils

import "strings"

var jsDateKeys = []string{
	"yyyy",
	"yy",
	"MM",
	"M",
	"dd",
	"d",
	"HH",
	"H",
	"hh",
	"h",
	"mm",
	"m",
	"ss",
	"s",
}
var goDateKeys = []string{
	"2006",
	"06",
	"01",
	"1",
	"02",
	"2",
	"15",
	"15",
	"03",
	"3",
	"04",
	"4",
	"05",
	"5",
}

func ConvertDateFormat(jsDateFormat string) string {
	goDateFormat := jsDateFormat

	for i, jsF := range jsDateKeys {
		goDateFormat = strings.ReplaceAll(goDateFormat, jsF, goDateKeys[i])
	}

	return goDateFormat
}
