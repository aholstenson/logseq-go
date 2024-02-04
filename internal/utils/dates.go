package utils

import "strings"

var jsDateKeys = []string{
	"yyyy",
	"yy",
	"MMM",
	"MM",
	"M",
	"do",
	"dd",
	"d",
	"EEE",
	"EEEE",
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
	"Jan",
	"01",
	"1",
	"2", // do isn't directly supported by Go - so default to 2
	"02",
	"2",
	"Mon",
	"Monday",
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
