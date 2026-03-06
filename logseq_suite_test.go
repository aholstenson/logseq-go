package logseq_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLogseq(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logseq Suite")
}
