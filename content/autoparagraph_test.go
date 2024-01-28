package content_test

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Autoparagaph", func() {
	It("single text node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello world"),
		})

		Expect(nodes).To(EqualsNodes(content.NewParagraph(
			content.NewText("Hello world"),
		)))
	})

	It("multiple text nodes", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewText("world"),
		})

		Expect(nodes).To(EqualsNodes(content.NewParagraph(
			content.NewText("Hello"),
			content.NewText("world"),
		)))
	})

	It("text node interrupted by block node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
		))
	})

	It("text node interrupted by block node followed by text node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
			content.NewText("again"),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
			content.NewParagraph(
				content.NewText("again"),
			),
		))
	})

	It("text node interrupted by block node followed by text node interrupted by block node", func() {
		nodes := content.AddAutomaticParagraphs([]content.Node{
			content.NewText("Hello"),
			content.NewHeading(1, content.NewText("world")),
			content.NewText("again"),
			content.NewParagraph(content.NewText("and again")),
		})

		Expect(nodes).To(EqualsNodes(
			content.NewParagraph(
				content.NewText("Hello"),
			),
			content.NewHeading(1, content.NewText("world")),
			content.NewParagraph(
				content.NewText("again"),
			),
			content.NewParagraph(
				content.NewText("and again"),
			),
		))
	})
})

type equalsNodes struct {
	Expected []content.Node
}

func EqualsNodes(expected ...content.Node) types.GomegaMatcher {
	return &equalsNodes{
		Expected: expected,
	}
}

func (matcher *equalsNodes) Match(actual interface{}) (bool, error) {
	var nodes []content.Node
	if node, ok := actual.(content.Node); ok {
		nodes = []content.Node{node}
	} else if nodeSlice, ok := actual.([]content.Node); ok {
		nodes = nodeSlice
	} else {
		return false, nil
	}

	if len(nodes) != len(matcher.Expected) {
		return false, nil
	}

	for i, node := range nodes {
		if content.Debug(node) != content.Debug(matcher.Expected[i]) {
			return false, nil
		}
	}

	return true, nil
}

func (matcher *equalsNodes) FailureMessage(actual interface{}) (message string) {
	if node, ok := actual.([]content.Node); ok {
		return format.Message(gomegaNodeSlice(node), "to equal", gomegaNodeSlice(matcher.Expected))
	}

	return format.Message(actual, "to equal", gomegaNodeSlice(matcher.Expected))
}

func (matcher *equalsNodes) NegatedFailureMessage(actual interface{}) (message string) {
	if node, ok := actual.([]content.Node); ok {
		return format.Message(gomegaNodeSlice(node), "not to equal", gomegaNodeSlice(matcher.Expected))
	}

	return format.Message(actual, "not to equal", gomegaNodeSlice(matcher.Expected))
}

var _ types.GomegaMatcher = &equalsNodes{}

type gomegaNodeSlice []content.Node

func (n gomegaNodeSlice) GomegaString() string {
	var s strings.Builder
	s.WriteString("[]content.Node{\n")
	for _, node := range n {
		s.WriteString(content.Debug(node))
	}
	s.WriteString("}")
	return s.String()
}

var _ format.GomegaStringer = gomegaNodeSlice{}
