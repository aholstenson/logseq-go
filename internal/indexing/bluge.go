package indexing

import (
	"context"
	"fmt"
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

func (i *BlugeIndex) DeleteDocument(ctx context.Context, subPath string) error {
	err := i.indexDelete(subPath)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	return nil
}

func (i *BlugeIndex) IndexDocument(ctx context.Context, doc *Document) error {
	blugeDoc, err := i.pageToDocument(doc)
	if err != nil {
		return err
	}

	err = i.indexUpdate(blugeDoc)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
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

func (i *BlugeIndex) pageToDocument(doc *Document) (*bluge.Document, error) {
	blugeDoc := bluge.NewDocument(doc.SubPath).
		AddField(bluge.NewDateTimeField("lastModified", doc.LastModified).StoreValue())

	switch doc.Type {
	case DocumentTypePage:
		blugeDoc.AddField(bluge.NewKeywordField("type", "page").StoreValue())
		blugeDoc.AddField(bluge.NewTextField("title", doc.Title).StoreValue())
	case DocumentTypeJournal:
		blugeDoc.AddField(bluge.NewKeywordField("type", "journal").StoreValue())
		blugeDoc.AddField(bluge.NewDateTimeField("date", doc.Date).StoreValue())
	}

	if len(doc.Blocks) > 0 {
		props := doc.Blocks[0].Properties()
		i.transferProperties(blugeDoc, props)
		i.transferRefs(blugeDoc, "pages", doc.Blocks[0])
	}

	var fullText strings.Builder
	for _, block := range doc.Blocks {
		plainText0(block.Children(), &fullText)
		fullText.WriteString("\n\n")
	}

	blugeDoc.AddField(bluge.NewTextField("content", fullText.String()))
	return blugeDoc, nil
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
		doc.AddField(bluge.NewTextField(field+":ref", ref.(content.PageRef).To()))

		if hashtag, ok := ref.(*content.Hashtag); ok {
			doc.AddField(bluge.NewTextField(field+":tag", hashtag.To()))
		}
	}
}

func (i *BlugeIndex) ListDocuments(ctx context.Context, q Query) (Iterator[*Document], error) {
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

	it, err := reader.Search(ctx, bluge.NewAllMatches(queryWithOnlyDocs))
	return &blugeIterator{
		it: it,
	}, err
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
			builder.WriteString(n.To())
		case *content.PageLink:
			builder.WriteString(n.To())
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

		return bluge.NewMatchQuery(query.text).SetField(query.field)
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

type blugeIterator struct {
	it search.DocumentMatchIterator
}

func (i *blugeIterator) Next() (*Document, error) {
	match, err := i.it.Next()
	if err != nil {
		return nil, err
	}

	if match == nil {
		return nil, nil
	}

	var doc Document
	match.VisitStoredFields(func(field string, value []byte) bool {
		switch field {
		case "_id":
			doc.SubPath = string(value)
		case "lastModified":
			t, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			doc.LastModified = t
		case "type":
			switch string(value) {
			case "page":
				doc.Type = DocumentTypePage
			case "journal":
				doc.Type = DocumentTypeJournal
			}
		case "title":
			doc.Title = string(value)
		case "date":
			t, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			doc.Date = t
		}

		return true
	})

	return &doc, nil
}
