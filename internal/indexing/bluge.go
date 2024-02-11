package indexing

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aholstenson/logseq-go/content"
	"github.com/aholstenson/logseq-go/internal/utils"
	"github.com/blugelabs/bluge"
	"github.com/blugelabs/bluge/index"
	"github.com/blugelabs/bluge/search"
)

type idTerm string

func (i idTerm) Field() string {
	return "_id"
}

func (i idTerm) Term() []byte {
	return []byte(i)
}

type BlugeIndex struct {
	mu sync.Mutex

	writer *bluge.Writer

	currentReader *bluge.Reader

	currentBatch     *index.Batch
	currentBatchSize int
}

func NewBlugeIndex(graphConfig *utils.GraphConfig, indexDirectory string) (*BlugeIndex, error) {
	var config bluge.Config
	if indexDirectory == "" {
		config = bluge.InMemoryOnlyConfig()
	} else {
		config = bluge.DefaultConfig(indexDirectory)
	}

	writer, err := bluge.OpenWriter(config)
	if err != nil {
		return nil, fmt.Errorf("error opening index writer: %w", err)
	}

	return &BlugeIndex{
		writer: writer,
	}, nil
}

func (i *BlugeIndex) Close() error {
	return i.writer.Close()
}

func (i *BlugeIndex) Sync() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.currentBatch == nil {
		return nil
	}

	// Apply the current batch
	err := i.writer.Batch(i.currentBatch)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	i.currentBatch = nil
	i.currentBatchSize = 0

	if i.currentReader != nil {
		// Close the current reader and allow it to be opened again
		err = i.currentReader.Close()
		if err != nil {
			return fmt.Errorf("error closing index reader: %w", err)
		}
		i.currentReader = nil
	}

	return nil
}

func (i *BlugeIndex) reader() (*bluge.Reader, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.currentReader == nil {
		reader, err := i.writer.Reader()
		if err != nil {
			return nil, fmt.Errorf("error opening index reader: %w", err)
		}

		i.currentReader = reader
	}

	return i.currentReader, nil
}

func (i *BlugeIndex) indexUpdate(doc *bluge.Document) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.currentBatch == nil {
		i.currentBatch = &index.Batch{}
	}

	i.currentBatch.Update(doc.ID(), doc)
	i.currentBatchSize++

	if i.currentBatchSize >= 1000 {
		err := i.writer.Batch(i.currentBatch)
		if err != nil {
			return fmt.Errorf("error updating index: %w", err)
		}

		i.currentBatch = nil
		i.currentBatchSize = 0
	}

	return nil
}

func (i *BlugeIndex) indexDelete(id string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.currentBatch == nil {
		i.currentBatch = &index.Batch{}
	}

	i.currentBatch.Delete(idTerm(id))
	i.currentBatchSize++

	if i.currentBatchSize >= 1000 {
		err := i.writer.Batch(i.currentBatch)
		if err != nil {
			return fmt.Errorf("error updating index: %w", err)
		}

		i.currentBatch = nil
		i.currentBatchSize = 0
	}

	return nil
}

func (i *BlugeIndex) DeletePage(ctx context.Context, subPath string) error {
	err := i.indexDelete(subPath)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	// Delete all of the blocks associated with the page
	idSet, err := i.getBlocks(ctx, subPath)
	if err != nil {
		return fmt.Errorf("error getting blocks: %w", err)
	}

	for id := range idSet {
		err = i.indexDelete(id)
		if err != nil {
			return fmt.Errorf("error updating index: %w", err)
		}
	}

	return nil
}

func (i *BlugeIndex) IndexPage(ctx context.Context, page *Page) error {
	blugeDoc, err := i.pageToDocument(page)
	if err != nil {
		return err
	}

	err = i.indexUpdate(blugeDoc)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	err = i.indexBlocks(ctx, page)
	if err != nil {
		return fmt.Errorf("error indexing blocks: %w", err)
	}

	return nil
}

func (i *BlugeIndex) GetLastModified(ctx context.Context, subPath string) (time.Time, error) {
	reader, err := i.reader()
	if err != nil {
		return time.Time{}, err
	}

	it, err := reader.Search(ctx, bluge.NewTopNSearch(1, bluge.NewTermQuery(subPath).SetField("_id")))
	if err != nil {
		return time.Time{}, fmt.Errorf("error searching index: %w", err)
	}

	match, err := it.Next()
	if err != nil {
		return time.Time{}, fmt.Errorf("error getting next match: %w", err)
	}

	if match == nil {
		return time.Time{}, nil
	}

	var lastModified time.Time
	match.VisitStoredFields(func(field string, value []byte) bool {
		if field == "lastModified" {
			time, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			lastModified = time
			return false
		}

		return true
	})

	return lastModified, nil
}

func (i *BlugeIndex) pageToDocument(doc *Page) (*bluge.Document, error) {
	blugeDoc := bluge.NewDocument(doc.SubPath).
		AddField(bluge.NewDateTimeField("lastModified", doc.LastModified).StoreValue())

	switch doc.Type {
	case PageTypeDedicated:
		blugeDoc.AddField(bluge.NewKeywordField("type", "page").StoreValue())
		blugeDoc.AddField(bluge.NewTextField("title", doc.Title).StoreValue())
	case PageTypeJournal:
		blugeDoc.AddField(bluge.NewKeywordField("type", "journal").StoreValue())
		blugeDoc.AddField(bluge.NewDateTimeField("date", doc.Date).StoreValue())
	}

	if len(doc.Blocks) > 0 {
		props := doc.Blocks[0].Properties()
		i.transferProperties(blugeDoc, props)
		i.transferRefs(blugeDoc, "pages", doc.Blocks[0])

		preview := generatePreview(doc.Blocks[0].Children())
		blugeDoc.AddField(bluge.NewKeywordField("preview", preview).StoreValue())
	}

	var fullText strings.Builder
	for idx, block := range doc.Blocks {
		i.transferLinks(blugeDoc, block)

		if idx > 0 {
			fullText.WriteString("\n\n")
		}

		plainText0(block.Children(), &fullText)
	}
	blugeDoc.AddField(bluge.NewTextField("content", fullText.String()))

	return blugeDoc, nil
}

func (i *BlugeIndex) indexBlocks(ctx context.Context, page *Page) error {
	idSet, err := i.getBlocks(ctx, page.SubPath)
	if err != nil {
		return fmt.Errorf("error getting blocks: %w", err)
	}

	// Index the new blocks
	for _, block := range page.Blocks {
		err = i.indexBlock(idSet, page, block)
		if err != nil {
			return err
		}
	}

	// Remove any blocks that are no longer present
	for id := range idSet {
		err = i.indexDelete(id)
		if err != nil {
			return fmt.Errorf("error updating index: %w", err)
		}
	}

	return nil
}

func (i *BlugeIndex) getBlocks(ctx context.Context, pagePath string) (map[string]struct{}, error) {
	reader, err := i.reader()
	if err != nil {
		return nil, err
	}

	// To enable us to remove any indexed blocks that are no longer present
	// on the page we search for blocks and keep track of their IDs.
	it, err := reader.Search(ctx, bluge.NewAllMatches(
		bluge.NewTermQuery(pagePath).SetField("page"),
	))
	if err != nil {
		return nil, fmt.Errorf("error searching index: %w", err)
	}

	idSet := make(map[string]struct{})
	for {
		match, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("error getting next match: %w", err)
		}

		if match == nil {
			break
		}

		match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				idSet[string(value)] = struct{}{}
				return false
			}

			return true
		})
	}

	return idSet, nil
}

func (i *BlugeIndex) indexBlock(idSet map[string]struct{}, page *Page, block *content.Block) error {
	id := blockID(page, block)

	blugeDoc, err := i.blockToDocument(page, id, block)
	if err != nil {
		return err
	}

	err = i.indexUpdate(blugeDoc)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	delete(idSet, id)

	for _, child := range block.Blocks() {
		err = i.indexBlock(idSet, page, child)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *BlugeIndex) blockToDocument(page *Page, id string, block *content.Block) (*bluge.Document, error) {
	blugeDoc := bluge.NewDocument(id).
		AddField(bluge.NewKeywordField("type", "block").StoreValue()).
		AddField(bluge.NewKeywordField("page", page.SubPath).StoreValue())

	if id := block.ID(); id != "" {
		blugeDoc.AddField(bluge.NewKeywordField("id", id).StoreValue())
	}

	props := block.Properties()
	i.transferProperties(blugeDoc, props)
	i.transferRefs(blugeDoc, "pages", block)
	i.transferLinks(blugeDoc, block)

	var fullText strings.Builder
	plainText0(block.Content(), &fullText)
	blugeDoc.AddField(bluge.NewTextField("content", fullText.String()))

	preview := generatePreview(block.Content())
	blugeDoc.AddField(bluge.NewTextField("preview", preview).StoreValue())

	return blugeDoc, nil
}

// blockID returns a semi-stable ID based on the location of the block on the
// page.
func blockID(page *Page, block *content.Block) string {
	var path strings.Builder
	path.WriteString(page.SubPath)

	current := block
	for current != nil {
		idx := 0
		for sibling := current.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
			idx++
		}

		path.WriteRune(':')
		path.WriteString(strconv.Itoa(idx))

		// Move up the hierarchy
		parent := current.Parent()
		if parent == nil {
			break
		}
		current = parent.(*content.Block)
	}

	return path.String()
}

func (i *BlugeIndex) transferProperties(doc *bluge.Document, properties *content.Properties) {
	for _, node := range properties.Children() {
		prop, ok := node.(*content.Property)
		if !ok {
			continue
		}

		i.transferRefs(doc, "prop:"+prop.Name, prop)

		s := plainText(prop.Children())
		if s == "" {
			continue
		}

		doc.AddField(bluge.NewTextField("prop:"+prop.Name+":text", s))
		doc.AddField(bluge.NewKeywordField("prop:"+prop.Name+":value", s))
	}
}

func (i *BlugeIndex) transferRefs(doc *bluge.Document, field string, root content.HasChildren) {
	refs := root.Children().FilterDeep(content.IsOfType[content.PageRef]())
	for _, ref := range refs {
		doc.AddField(bluge.NewKeywordField(field+":ref", ref.(content.PageRef).GetTo()))

		if hashtag, ok := ref.(*content.Hashtag); ok {
			doc.AddField(bluge.NewKeywordField(field+":tag", hashtag.GetTo()))
		}
	}
}

func (i *BlugeIndex) transferLinks(doc *bluge.Document, root content.HasChildren) {
	links := root.Children().FilterDeep(content.IsOfType[content.HasLinkURL]())
	for _, link := range links {
		doc.AddField(bluge.NewKeywordField("link", link.(content.HasLinkURL).GetURL()))
	}
}

func (i *BlugeIndex) SearchPages(ctx context.Context, q Query, opts SearchOptions) (SearchResults[*Page], error) {
	if opts.Size <= 0 {
		opts.Size = 10
	}

	mappedQuery := mapQuery(q)

	reader, err := i.reader()
	if err != nil {
		return nil, err
	}

	queryWithOnlyDocs := bluge.NewBooleanQuery().
		AddMust(bluge.NewBooleanQuery().
			AddShould(bluge.NewTermQuery("page").SetField("type")).
			AddShould(bluge.NewTermQuery("journal").SetField("type")),
		).
		AddMust(mappedQuery)

	req := bluge.NewTopNSearch(opts.Size, queryWithOnlyDocs).
		WithStandardAggregations().
		SetFrom(opts.From)

	i.transferSortBy(opts, req)

	it, err := reader.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error searching index: %w", err)
	}

	return newBlugeSearchResults(ctx, it, mapMatchToPage)
}

func (i *BlugeIndex) SearchBlocks(ctx context.Context, q Query, opts SearchOptions) (SearchResults[*Block], error) {
	if opts.Size <= 0 {
		opts.Size = 10
	}

	mappedQuery := mapQuery(q)

	reader, err := i.reader()
	if err != nil {
		return nil, err
	}

	queryWithOnlyBlocks := bluge.NewBooleanQuery().
		AddMust(bluge.NewTermQuery("block").SetField("type")).
		AddMust(mappedQuery)

	req := bluge.NewTopNSearch(opts.Size, queryWithOnlyBlocks).
		WithStandardAggregations().
		SetFrom(opts.From)

	i.transferSortBy(opts, req)

	it, err := reader.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error searching index: %w", err)
	}

	return newBlugeSearchResults(ctx, it, mapMatchToBlock)
}

func (*BlugeIndex) transferSortBy(opts SearchOptions, req *bluge.TopNSearch) {
	if len(opts.SortBy) > 0 {
		var sortOrder search.SortOrder

		for _, sortField := range opts.SortBy {
			sortBy := search.SortBy(search.Field(sortField.Field))
			if !sortField.Asc {
				sortBy.Desc()
			}

			sortOrder = append(sortOrder, sortBy)
		}

		req.SortByCustom(sortOrder)
	}
}

// generatePreview takes a list of nodes and generates a preview of the content.
// Previews are intended for the user in search results, so we look for the first
// paragraph, list, blockquote or code block and use that as the preview.
func generatePreview(nodes content.NodeList) string {
	for _, node := range nodes {
		switch n := node.(type) {
		case *content.Paragraph:
			return plainText(n.Children())
		case *content.List:
			return plainText(n.Children())
		case *content.Blockquote:
			return plainText(n.Children())
		case *content.CodeBlock:
			return n.Code
		}
	}

	return ""
}

func plainText(nodes content.NodeList) string {
	var builder strings.Builder
	plainText0(nodes, &builder)
	return strings.TrimSpace(builder.String())
}

func plainText0(nodes content.NodeList, builder *strings.Builder) {
	for _, node := range nodes {
		switch n := node.(type) {
		case *content.Text:
			builder.WriteString(n.Value)
			if n.SoftLineBreak || n.HardLineBreak {
				builder.WriteRune('\n')
			}
		case *content.Hashtag:
			builder.WriteString("#")
			builder.WriteString(n.To)
		case *content.PageLink:
			builder.WriteString(n.To)
		case *content.CodeSpan:
			builder.WriteString(n.Value)
		case *content.CodeBlock:
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}

			builder.WriteString(n.Code)
		case *content.Properties:
			// Skip properties
		case content.HasChildren:
			if _, blockNode := n.(content.BlockNode); blockNode && builder.Len() > 0 {
				builder.WriteString("\n\n")
			}

			plainText0(n.Children(), builder)
		}
	}
}

func mapQuery(q Query) bluge.Query {
	switch query := q.(type) {
	case *all:
		return bluge.NewMatchAllQuery()
	case *none:
		return bluge.NewMatchNoneQuery()
	case *and:
		bq := bluge.NewBooleanQuery()
		for _, sub := range query.clauses {
			bq = bq.AddMust(mapQuery(sub))
		}
		return bq
	case *or:
		bq := bluge.NewBooleanQuery()
		for _, sub := range query.clauses {
			bq = bq.AddShould(mapQuery(sub))
		}
		return bq
	case *not:
		return bluge.NewBooleanQuery().AddMustNot(mapQuery(query.clause))
	case *fieldMatches:
		if strings.HasPrefix(query.field, "prop:") {
			return bluge.NewMatchQuery(query.text).SetField(query.field + ":text")
		}

		return bluge.NewMatchQuery(query.text).SetOperator(bluge.MatchQueryOperatorAnd).SetField(query.field)
	case *fieldEquals:
		if strings.HasPrefix(query.field, "prop:") {
			return bluge.NewTermQuery(query.value).SetField(query.field + ":value")
		}

		return bluge.NewTermQuery(query.value).SetField(query.field)
	case *fieldRefs:
		if query.tag {
			return bluge.NewTermQuery(query.target).SetField(query.field + ":tag")
		}

		return bluge.NewTermQuery(query.target).SetField(query.field + ":ref")
	default:
		return bluge.NewMatchNoneQuery()
	}
}

type blugeSearchResults[D any] struct {
	mu sync.Mutex

	count   int
	results []D
}

func newBlugeSearchResults[V any](ctx context.Context, it search.DocumentMatchIterator, mapper func(*search.DocumentMatch) V) (*blugeSearchResults[V], error) {
	results := make([]V, 0)
	for {
		match, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("error getting next match: %w", err)
		}

		if match == nil {
			break
		}

		doc := mapper(match)
		results = append(results, doc)
	}

	return &blugeSearchResults[V]{
		count:   int(it.Aggregations().Count()),
		results: results,
	}, nil
}

func (r *blugeSearchResults[V]) Size() int {
	return len(r.results)
}

func (r *blugeSearchResults[V]) Count() int {
	return r.count
}

func (r *blugeSearchResults[V]) Results() []V {
	return r.results
}

var _ SearchResults[*Page] = &blugeSearchResults[*Page]{}

func mapMatchToPage(match *search.DocumentMatch) *Page {
	page := &Page{}

	match.VisitStoredFields(func(field string, value []byte) bool {
		switch field {
		case "_id":
			page.SubPath = string(value)
		case "lastModified":
			t, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			page.LastModified = t
		case "type":
			switch string(value) {
			case "page":
				page.Type = PageTypeDedicated
			case "journal":
				page.Type = PageTypeJournal
			}
		case "title":
			page.Title = string(value)
		case "date":
			t, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			page.Date = t
		}

		return true
	})

	return page
}

func mapMatchToBlock(match *search.DocumentMatch) *Block {
	block := &Block{}

	match.VisitStoredFields(func(field string, value []byte) bool {
		switch field {
		case "_id":
			// The ID of the block is the sub path of the page with the location in reverse
			// order appended to it.
			location := make([]int, 0)
			for _, part := range strings.Split(string(value), ":") {
				idx, err := strconv.Atoi(part)
				if err != nil {
					break
				}

				location = append(location, idx)
			}

			// Reverse the location
			for i, j := 0, len(location)-1; i < j; i, j = i+1, j-1 {
				location[i], location[j] = location[j], location[i]
			}

			block.Location = location
		case "page":
			block.PageSubPath = string(value)
		case "id":
			block.ID = string(value)
		case "preview":
			block.Preview = string(value)
		}

		return true
	})

	return block
}
