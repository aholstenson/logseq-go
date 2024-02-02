package tests

import (
	"strings"

	"github.com/aholstenson/logseq-go/content"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type equalNode struct {
	Expected content.Node
}

func EqualNode(expected content.Node) types.GomegaMatcher {
	return &equalNode{
		Expected: expected,
	}
}

func (matcher *equalNode) Match(actual interface{}) (bool, error) {
	if node, ok := actual.(content.Node); ok {
		return content.Debug(node) == content.Debug(matcher.Expected), nil
	}

	return false, nil
}

func (matcher *equalNode) FailureMessage(actual interface{}) (message string) {
	if node, ok := actual.(content.Node); ok {
		return format.Message(gomegaNode{Node: node}, "to equal", gomegaNode{Node: matcher.Expected})
	}

	return format.Message(actual, "to equal", gomegaNode{Node: matcher.Expected})
}

func (matcher *equalNode) NegatedFailureMessage(actual interface{}) (message string) {
	if node, ok := actual.(content.Node); ok {
		return format.Message(gomegaNode{Node: node}, "not to equal", gomegaNode{Node: matcher.Expected})
	}

	return format.Message(actual, "not to equal", gomegaNode{Node: matcher.Expected})
}

var _ types.GomegaMatcher = &equalNode{}

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
	} else if nodeList, ok := actual.(content.NodeList); ok {
		nodes = nodeList
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
	if nodeList, ok := actual.(content.NodeList); ok {
		return format.Message(gomegaNodeSlice(nodeList), "to equal", gomegaNodeSlice(matcher.Expected))
	} else if node, ok := actual.([]content.Node); ok {
		return format.Message(gomegaNodeSlice(node), "to equal", gomegaNodeSlice(matcher.Expected))
	} else if node, ok := actual.(content.Node); ok {
		return format.Message(gomegaNode{Node: node}, "to equal", gomegaNodeSlice(matcher.Expected))
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

type gomegaNode struct {
	content.Node
}

func (n gomegaNode) GomegaString() string {
	return content.Debug(content.Node(n))
}

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
