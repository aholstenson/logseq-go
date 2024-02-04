package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

type FilenameFormat string

const (
	FilenameFormatTripleLowbar FilenameFormat = "triple-lowbar"
)

func TitleToFilename(format FilenameFormat, title string) (string, error) {
	switch format {
	case FilenameFormatTripleLowbar, "":
		// The default for Logseq, / are replaced with ___, percent encoding
		// for other invalid characters.
		return triLbFileNameSanity(title), nil
	}

	return "", fmt.Errorf("unknown file name format: %s", format)
}

var urlEncodedPattern = regexp.MustCompile("(?i)%[0-9a-f]{2}")
var reservedCharsPattern = regexp.MustCompile(`[:\\*\?"<>|#\\]+`)
var windowsReservedFileBodies = []string{
	"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6",
	"COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6",
	"LPT7", "LPT8", "LPT9",
}

func includeReservedChars(s string) bool {
	return reservedCharsPattern.MatchString(s)
}

func encodeURLPercent(input string) string {
	return strings.ReplaceAll(input, "%", "%25")
}

func escapeNamespaceSlashesAndMultilowbars(s string) string {
	s = strings.ReplaceAll(s, "___", "%5F%5F%5F")
	s = strings.ReplaceAll(s, "_/", "%5F/")
	s = strings.ReplaceAll(s, "/_", "/%5F")
	s = strings.ReplaceAll(s, "/", "___")
	return s
}

func escapeWindowsReservedFileBodies(fileBody string) string {
	if contains(windowsReservedFileBodies, fileBody) || strings.HasSuffix(fileBody, ".") {
		return fileBody + "/"
	}
	return fileBody
}

func urlEncodeFileName(fileName string) string {
	fileName = url.QueryEscape(fileName)
	fileName = strings.ReplaceAll(fileName, "*", "%2A")
	return fileName
}

func pageNameSanity(pageName string) string {
	pageName = removeBoundarySlashes(pageName)
	pageName = pathNormalize(pageName)
	return pageName
}

func removeBoundarySlashes(s string) string {
	if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "/") {
		s = s[:len(s)-1]
	}
	return s
}

func pathNormalize(s string) string {
	return norm.NFC.String(s)
}

func triLbFileNameSanity(title string) string {
	title = pageNameSanity(title)
	title = urlEncodedPattern.ReplaceAllStringFunc(title, encodeURLPercent)
	title = reservedCharsPattern.ReplaceAllStringFunc(title, urlEncodeFileName)
	title = escapeWindowsReservedFileBodies(title)
	title = escapeNamespaceSlashesAndMultilowbars(title)
	return title
}

func contains(arr []string, value string) bool {
	for _, v := range arr {
		if v == value {
			return true
		}
	}
	return false
}

func FilenameToTitle(format FilenameFormat, filename string) (string, error) {
	switch format {
	case FilenameFormatTripleLowbar, "":
		// Reverse the transformations done in triLbFileNameSanity
		title := filename
		title = unescapeNamespaceSlashesAndMultilowbars(title)
		title = unescapeWindowsReservedFileBodies(title)
		title = urlEncodedPattern.ReplaceAllStringFunc(title, decodeURLPercent)
		title = pathNormalize(title)
		return title, nil
	}

	return "", fmt.Errorf("unknown file name format: %s", format)
}

func unescapeNamespaceSlashesAndMultilowbars(s string) string {
	s = strings.ReplaceAll(s, "___", "/")
	s = strings.ReplaceAll(s, "%5F%5F%5F", "___")
	s = strings.ReplaceAll(s, "%5F/", "_/")
	s = strings.ReplaceAll(s, "/%5F", "/_")
	return s
}

func unescapeWindowsReservedFileBodies(fileBody string) string {
	if strings.HasSuffix(fileBody, "/") {
		fileBody = fileBody[:len(fileBody)-1]
	}
	return fileBody
}

func decodeURLPercent(input string) string {
	decoded, err := url.QueryUnescape(input)
	if err != nil {
		return input
	}
	return decoded
}
