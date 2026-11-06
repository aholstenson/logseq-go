package content

import "sort"

// NestedPageRef is a page link inside the title of another page link or tag,
// such as `b` in `[[a [[b]]]]`. Logseq refers to the page of the whole title
// and to each page that is linked inside it, at any depth.
//
// A NestedPageRef is not a node of the tree. PageReferences returns one for
// each page link in the title. It reads and changes the title of the outer
// reference, so pointing it at another page also changes the outer title:
// `b` in `[[a [[b]]]]` pointed at `c` gives `[[a [[c]]]]`.
type NestedPageRef struct {
	baseNode

	// outer is the reference that has the link in its title.
	outer PageRef

	// index is the position of the link among the links in the title of
	// outer, in the order they start.
	index int
}

// Outer gets the page link or tag that has this link in its title.
func (r *NestedPageRef) Outer() PageRef {
	return r.outer
}

// Parent gets the parent of the outer reference, as this reference is not in
// the tree itself.
func (r *NestedPageRef) Parent() HasChildren {
	return r.outer.Parent()
}

func (r *NestedPageRef) GetTo() string {
	title := r.outer.GetTo()
	spans := nestedPageLinks(title)
	if r.index >= len(spans) {
		return ""
	}

	span := spans[r.index]
	return title[span[0]:span[1]]
}

func (r *NestedPageRef) SetTo(target string) {
	title := r.outer.GetTo()
	spans := nestedPageLinks(title)
	if r.index >= len(spans) {
		return
	}

	span := spans[r.index]
	r.outer.SetTo(title[:span[0]] + target + title[span[1]:])
}

func (r *NestedPageRef) debug(p *debugPrinter) {
	p.StartType("NestedPageRef")
	p.Field("to", r.GetTo())
	p.EndType()
}

func (r *NestedPageRef) isPageRef() {}

var _ PageRef = (*NestedPageRef)(nil)

// nestedPageRefsOf gets a NestedPageRef for each page link in the title of
// the reference. Only page links and tags can have page links in their title.
func nestedPageRefsOf(ref PageRef) []Node {
	switch ref.(type) {
	case *PageLink, *Hashtag:
	default:
		return nil
	}

	spans := nestedPageLinks(ref.GetTo())
	refs := make([]Node, 0, len(spans))
	for i := range spans {
		refs = append(refs, &NestedPageRef{outer: ref, index: i})
	}

	return refs
}

// nestedPageLinks finds the page links in a page title, such as `b` in
// `a [[b]]`, at any depth, in the order they start. Each one is the start and
// the end of the title of the link. A `[[` without a `]]` is not a link.
func nestedPageLinks(title string) [][2]int {
	var spans [][2]int
	var open []int
	for i := 0; i+1 < len(title); i++ {
		if title[i] == '[' && title[i+1] == '[' {
			open = append(open, i+2)
			i++
		} else if title[i] == ']' && title[i+1] == ']' && len(open) > 0 {
			start := open[len(open)-1]
			open = open[:len(open)-1]
			spans = append(spans, [2]int{start, i})
			i++
		}
	}

	// A link is closed after the links inside it, so they are sorted to be in
	// the order they start.
	sort.Slice(spans, func(a, b int) bool {
		return spans[a][0] < spans[b][0]
	})

	return spans
}
