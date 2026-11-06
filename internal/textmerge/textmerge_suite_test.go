package textmerge

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTextMerge(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Text Merge Suite")
}
