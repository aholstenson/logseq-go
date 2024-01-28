package content

// AddAutomaticParagraphs wraps all nodes that are not blocks in paragraphs.
// This enables the user to create nodes without having to think too much about
// the structure of the document.
func AddAutomaticParagraphs(nodes []Node) []Node {
	var currentParagraph *Paragraph

	rewrittenNodes := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := node.(BlockNode); ok {
			if currentParagraph != nil {
				// There is an open paragraph, add it to the rewritten nodes.
				rewrittenNodes = append(rewrittenNodes, currentParagraph)
				currentParagraph = nil
			}

			rewrittenNodes = append(rewrittenNodes, node)
			continue
		}

		if currentParagraph == nil {
			currentParagraph = NewParagraph()
		}

		currentParagraph.AddChild(node)
	}

	if currentParagraph != nil {
		rewrittenNodes = append(rewrittenNodes, currentParagraph)
	}

	return rewrittenNodes
}
