package indexing_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIndexing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Indexing Suite")
}
