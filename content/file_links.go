package content

import "strings"

// FileLinkRef is the page reference that Logseq reads from a link to a
// Markdown file, such as `[My page](../pages/my_page.md)`. Logseq refers to
// the page named by the label of the link, here `My page`, and not to the page
// of the file.
//
// A FileLinkRef is not a node of the tree. PageReferences returns one for each
// link with a label that names a page. It reads and changes the label of the
// link.
//
// Logseq does not change these links when it renames a page, so renaming
// leaves them as they are.
type FileLinkRef struct {
	baseNode

	// link is the link that this reference is read from.
	link *Link
}

// Link gets the link that this reference is read from.
func (r *FileLinkRef) Link() *Link {
	return r.link
}

// Parent gets the parent of the link, as this reference is not in the tree
// itself.
func (r *FileLinkRef) Parent() HasChildren {
	return r.link.Parent()
}

// GetTo gets the title of the page, which is the label of the link.
func (r *FileLinkRef) GetTo() string {
	title, _ := fileLinkPageTitle(r.link)
	return title
}

// SetTo changes the label of the link to the title of another page. The URL
// of the link does not change.
func (r *FileLinkRef) SetTo(target string) {
	r.link.SetChildren(NewText(target))
}

func (r *FileLinkRef) debug(p *debugPrinter) {
	p.StartType("FileLinkRef")
	p.Field("to", r.GetTo())
	p.EndType()
}

func (r *FileLinkRef) isPageRef() {}

var _ PageRef = (*FileLinkRef)(nil)

// fileLinkRefOf gets the reference that Logseq reads from a link, or nil if
// Logseq reads no reference from it. Logseq reads no reference from a link in
// the value of a property.
func fileLinkRefOf(link *Link) Node {
	if _, ok := fileLinkPageTitle(link); !ok {
		return nil
	}

	for parent := link.Parent(); parent != nil; parent = parent.Parent() {
		if _, ok := parent.(*Property); ok {
			return nil
		}
	}

	return &FileLinkRef{link: link}
}

// fileLinkPageTitle gets the title of the page that Logseq refers to from a
// link to a Markdown file. Logseq takes the first part of the label: the text
// before the first code span, or the code of a code span at the start, such
// as `x` in "[`x` y](a.md)".
//
// Logseq takes the text before a code span with its emphasis markers, such as
// `a *b*`. The node tree does not keep these markers, so a label with
// emphasis, a page link or other markup before the first code span gives no
// title. A blank title, or a label on more than one line, gives no title too,
// as Logseq does not read a reference or a link from them.
func fileLinkPageTitle(link *Link) (string, bool) {
	if !isMarkdownFileURL(link.URL) {
		return "", false
	}

	children := link.Children()
	if len(children) == 0 {
		return "", false
	}

	if code, ok := children[0].(*CodeSpan); ok {
		return code.Value, strings.TrimSpace(code.Value) != ""
	}

	var title strings.Builder
	for _, child := range children {
		if _, ok := child.(*CodeSpan); ok {
			break
		}

		text, ok := child.(*Text)
		if !ok || text.SoftLineBreak || text.HardLineBreak {
			return "", false
		}

		title.WriteString(text.Value)
	}

	if strings.TrimSpace(title.String()) == "" {
		return "", false
	}

	return title.String(), true
}

// isMarkdownFileURL checks if Logseq reads the URL of a link as a Markdown
// file. This is a URL that ends in `.md` or `.markdown`, in any case, and has
// no protocol with `://`. Thus `../pages/a.md` and `file:../pages/a.md` are
// files, but `https://x.com/a.md`, `file:///a.md` and `a.md#part` are not.
func isMarkdownFileURL(url string) bool {
	url = strings.TrimSpace(url)
	if colon := strings.IndexByte(url, ':'); colon >= 0 && strings.HasPrefix(url[colon+1:], "//") {
		return false
	}

	lower := strings.ToLower(url)
	return len(url) > 3 && (strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"))
}
