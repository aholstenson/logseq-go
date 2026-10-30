package indexing

import (
	"context"
	"fmt"
	"math"
	"regexp"
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

// indexVersion is the version of the documents and fields in the index.
// Increase it when the way pages or blocks are indexed changes, so that an
// index stored on disk is built again with the new fields. Without that the
// pages that did not change since they were indexed keep their old documents.
const indexVersion = 11

// versionDocID is the id of the document that stores the version of the
// index. It does not have a type, so searches for pages and blocks never
// return it.
const versionDocID = "_logseq-go:index-version"

type BlugeIndex struct {
	mu sync.Mutex

	writer *bluge.Writer

	currentReader *bluge.Reader

	currentBatch     *index.Batch
	currentBatchSize int

	// journalTitles reads the titles of journals, so that a property that
	// references a journal can be compared as a date. Nil if the format is
	// not known.
	journalTitles *utils.DateFormat
}

func NewBlugeIndex(graphConfig *utils.GraphConfig, indexDirectory string) (*BlugeIndex, error) {
	idx, err := newBlugeIndex(indexDirectory, indexVersion)
	if err != nil {
		return nil, err
	}

	if graphConfig != nil && graphConfig.JournalPageTitleFormat != "" {
		idx.journalTitles = utils.NewDateFormat(graphConfig.JournalPageTitleFormat)
	}
	return idx, nil
}

func newBlugeIndex(indexDirectory string, version int) (*BlugeIndex, error) {
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

	idx := &BlugeIndex{
		writer: writer,
	}

	if err := idx.ensureVersion(version); err != nil {
		writer.Close()
		return nil, err
	}

	return idx, nil
}

// ensureVersion empties the index if it was built with another version, or
// before the index stored a version, and then stores the given version. The
// pages are then indexed again when the graph is synced, as none of them are
// in the index any more.
func (i *BlugeIndex) ensureVersion(version int) error {
	reader, err := i.writer.Reader()
	if err != nil {
		return fmt.Errorf("error opening index reader: %w", err)
	}
	defer reader.Close()

	ctx := context.Background()
	it, err := reader.Search(ctx, bluge.NewTopNSearch(1, bluge.NewTermQuery(versionDocID).SetField("_id")))
	if err != nil {
		return fmt.Errorf("error reading index version: %w", err)
	}

	match, err := it.Next()
	if err != nil {
		return fmt.Errorf("error reading index version: %w", err)
	}

	stored := ""
	if match != nil {
		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "version" {
				stored = string(value)
				return false
			}

			return true
		})
		if err != nil {
			return fmt.Errorf("error reading index version: %w", err)
		}
	}

	current := strconv.Itoa(version)
	if stored == current {
		return nil
	}

	batch := index.NewBatch()

	it, err = reader.Search(ctx, bluge.NewAllMatches(bluge.NewMatchAllQuery()))
	if err != nil {
		return fmt.Errorf("error listing documents in index: %w", err)
	}

	for {
		match, err := it.Next()
		if err != nil {
			return fmt.Errorf("error listing documents in index: %w", err)
		}

		if match == nil {
			break
		}

		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				if id := string(value); id != versionDocID {
					batch.Delete(idTerm(id))
				}
				return false
			}

			return true
		})
		if err != nil {
			return fmt.Errorf("error listing documents in index: %w", err)
		}
	}

	versionDoc := bluge.NewDocument(versionDocID).
		AddField(bluge.NewKeywordField("version", current).StoreValue())
	batch.Update(versionDoc.ID(), versionDoc)

	if err := i.writer.Batch(batch); err != nil {
		return fmt.Errorf("error storing index version: %w", err)
	}

	return nil
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

func (i *BlugeIndex) GetFileState(ctx context.Context, subPath string) (FileState, error) {
	reader, err := i.reader()
	if err != nil {
		return FileState{}, err
	}

	it, err := reader.Search(ctx, bluge.NewTopNSearch(1, bluge.NewTermQuery(subPath).SetField("_id")))
	if err != nil {
		return FileState{}, fmt.Errorf("error searching index: %w", err)
	}

	match, err := it.Next()
	if err != nil {
		return FileState{}, fmt.Errorf("error getting next match: %w", err)
	}

	if match == nil {
		return FileState{}, nil
	}

	var state FileState
	err = match.VisitStoredFields(func(field string, value []byte) bool {
		switch field {
		case "lastModified":
			time, err := bluge.DecodeDateTime(value)
			if err != nil {
				return false
			}

			state.LastModified = time
		case "revision":
			state.Revision = string(value)
		}

		return true
	})
	if err != nil {
		return FileState{}, fmt.Errorf("error reading stored fields: %w", err)
	}

	return state, nil
}

func (i *BlugeIndex) PageSubPaths(ctx context.Context) ([]string, error) {
	reader, err := i.reader()
	if err != nil {
		return nil, err
	}

	query := bluge.NewBooleanQuery().
		AddShould(bluge.NewTermQuery("page").SetField("type")).
		AddShould(bluge.NewTermQuery("journal").SetField("type"))

	it, err := reader.Search(ctx, bluge.NewAllMatches(query))
	if err != nil {
		return nil, fmt.Errorf("error searching index: %w", err)
	}

	var subPaths []string
	for {
		match, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("error getting next match: %w", err)
		}

		if match == nil {
			break
		}

		err = match.VisitStoredFields(func(field string, value []byte) bool {
			if field == "_id" {
				subPaths = append(subPaths, string(value))
				return false
			}

			return true
		})
		if err != nil {
			return nil, fmt.Errorf("error reading stored fields: %w", err)
		}
	}

	return subPaths, nil
}

func (i *BlugeIndex) pageToDocument(doc *Page) (*bluge.Document, error) {
	blugeDoc := bluge.NewDocument(doc.SubPath).
		AddField(bluge.NewDateTimeField("lastModified", doc.LastModified).StoreValue())

	if doc.Revision != "" {
		blugeDoc.AddField(bluge.NewKeywordField("revision", doc.Revision).StoreValue())
	}

	switch doc.Type {
	case PageTypeDedicated:
		blugeDoc.AddField(bluge.NewKeywordField("type", "page").StoreValue())
		blugeDoc.AddField(bluge.NewTextField("title", doc.Title).StoreValue())

		// The namespaces of a page are indexed both as the one it is directly
		// in and as all of them, so that the pages of a namespace can be found
		// with or without the ones deeper in it.
		namespaces := utils.NamespacesOf(doc.Title)
		if len(namespaces) > 0 {
			blugeDoc.AddField(bluge.NewKeywordField("namespace", normalizeRef(namespaces[0])))

			for _, namespace := range namespaces {
				blugeDoc.AddField(bluge.NewKeywordField("namespaces", normalizeRef(namespace)))
			}
		}
	case PageTypeJournal:
		blugeDoc.AddField(bluge.NewKeywordField("type", "journal").StoreValue())
		blugeDoc.AddField(bluge.NewDateTimeField("date", doc.Date).StoreValue())
	}

	addTitleSort(blugeDoc, doc)

	// Aliases are matched the same way references are, so that a page can be
	// found via a title that only differs in case from the alias.
	for _, alias := range doc.Aliases {
		blugeDoc.AddField(bluge.NewKeywordField("alias:ref", normalizeRef(alias)))
	}

	if doc.Properties != nil {
		i.transferProperties(blugeDoc, doc.Properties, true)
		i.transferRefs(blugeDoc, "pages", doc.Properties.Children())

		// The tags are stored so that a query can find the pages that other
		// pages use as tags, as the index can not join pages with pages.
		if tags := doc.Properties.GetAsNode("tags"); tags != nil {
			for _, ref := range tags.Children().PageReferences() {
				blugeDoc.AddField(bluge.NewKeywordField("tags", normalizeRef(ref.(content.PageRef).GetTo())).StoreValue())
			}
		}
	}

	// References in the block a page opens with are references of the page
	// itself. The pre-block holding the content before the first bullet is part
	// of that opening, so it is taken together with the first bullet.
	for _, block := range doc.Blocks {
		i.transferRefs(blugeDoc, "pages", block.Children())

		if !block.IsPreBlock() {
			break
		}
	}

	// The preview is the first thing a reader would see on the page, so skip
	// past blocks that render as nothing, such as a pre-block of only
	// properties.
	for _, block := range doc.Blocks {
		preview := generatePreview(block.Children())
		if preview != "" {
			blugeDoc.AddField(bluge.NewKeywordField("preview", preview).StoreValue())
			break
		}
	}

	var fullText strings.Builder
	for idx, block := range doc.Blocks {
		i.transferLinks(blugeDoc, block.Children())

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

	// The path of every block starts at the page it is on.
	var pagePathRefs []string
	if page.Title != "" {
		pagePathRefs = []string{normalizeRef(page.Title)}
	}

	// Index the new blocks
	for idx, block := range page.Blocks {
		err = i.indexBlock(idSet, page, block, pagePathRefs, []int{idx})
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

// indexBlock indexes a block and the blocks below it. parentPathRefs is the
// path refs of the parent block, or of the page for a top-level block.
// location is the place of the block on the page, see utils.LocationOf.
func (i *BlugeIndex) indexBlock(idSet map[string]struct{}, page *Page, block *content.Block, parentPathRefs []string, location []int) error {
	id := blockID(page, block)
	pathRefs := pathRefsOf(parentPathRefs, block)

	blugeDoc, err := i.blockToDocument(page, id, block, pathRefs, location)
	if err != nil {
		return err
	}

	err = i.indexUpdate(blugeDoc)
	if err != nil {
		return fmt.Errorf("error updating index: %w", err)
	}

	delete(idSet, id)

	for idx, child := range block.Blocks() {
		// The full slice expression makes append copy the location, so that
		// sibling blocks do not write to the same array.
		childLocation := append(location[:len(location):len(location)], idx)
		err = i.indexBlock(idSet, page, child, pathRefs, childLocation)
		if err != nil {
			return err
		}
	}

	return nil
}

// pathRefsOf returns the path refs of a block, which is what Logseq calls the
// pages that a block is about: the page it is on, the pages its parent blocks
// reference and the pages it references itself. The titles are normalized
// with normalizeRef. parentPathRefs is the path refs of the parent block, or
// of the page for a top-level block.
func pathRefsOf(parentPathRefs []string, block *content.Block) []string {
	// The full slice expression makes append copy the parent refs, so that
	// sibling blocks do not write to the same array.
	refs := parentPathRefs[:len(parentPathRefs):len(parentPathRefs)]

	// Only the content is used, as the blocks below this block are not part
	// of its path.
	for _, ref := range block.Content().PageReferences() {
		title := normalizeRef(ref.(content.PageRef).GetTo())
		if !containsString(refs, title) {
			refs = append(refs, title)
		}
	}

	return refs
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}

func (i *BlugeIndex) blockToDocument(page *Page, id string, block *content.Block, pathRefs []string, location []int) (*bluge.Document, error) {
	blugeDoc := bluge.NewDocument(id).
		AddField(bluge.NewKeywordField("type", "block").StoreValue()).
		AddField(bluge.NewKeywordField("page", page.SubPath).StoreValue())

	for _, ref := range pathRefs {
		blugeDoc.AddField(bluge.NewKeywordField("path:ref", ref))
	}

	// Blocks on a journal get the date of the journal, in the same field as
	// the journal page, so that one query finds both.
	if page.Type == PageTypeJournal {
		blugeDoc.AddField(bluge.NewDateTimeField("date", page.Date))
	}

	addTitleSort(blugeDoc, page)

	if id := block.ID(); id != "" {
		blugeDoc.AddField(bluge.NewKeywordField("id", id).StoreValue())
	}

	// Blocks are found by their parent through the id of the parent, so only
	// the blocks below a parent with an id get the field.
	if parent, ok := block.Parent().(*content.Block); ok {
		if id := parent.ID(); id != "" {
			blugeDoc.AddField(bluge.NewKeywordField("parent", id))
		}
	}

	// The same is true for the blocks above the block at any depth.
	for parent, ok := block.Parent().(*content.Block); ok; parent, ok = parent.Parent().(*content.Block) {
		if id := parent.ID(); id != "" {
			blugeDoc.AddField(bluge.NewKeywordField("ancestor", id))
		}
	}

	// Blocks without an id are found by their place on the page instead. The
	// place of each block above this block is the start of its location.
	for depth := 1; depth < len(location); depth++ {
		blugeDoc.AddField(bluge.NewKeywordField("ancestor:location", formatLocation(location[:depth])))
	}
	if len(location) > 1 {
		blugeDoc.AddField(bluge.NewKeywordField("parent:location", formatLocation(location[:len(location)-1])))
	}

	// The depth is 1 for the blocks at the top of the page, which have the
	// page as their parent.
	blugeDoc.AddField(bluge.NewNumericField("depth", float64(len(location))))

	// Look up the properties without creating them, as indexing should not
	// modify the block.
	if props := block.FindProperties(); props != nil {
		i.transferProperties(blugeDoc, props, true)
	}
	// Only the content of the block itself is used, as the blocks below it
	// are indexed on their own. Logseq calls these the refs of a block, and
	// PathReferences matches on the refs of the parents as well.
	i.transferRefs(blugeDoc, "pages", block.Content())
	i.transferLinks(blugeDoc, block.Content())
	transferTask(blugeDoc, block)

	var fullText strings.Builder
	plainText0(block.Content(), &fullText)
	blugeDoc.AddField(bluge.NewTextField("content", fullText.String()))

	preview := generatePreview(block.Content())
	blugeDoc.AddField(bluge.NewTextField("preview", preview).StoreValue())

	return blugeDoc, nil
}

// transferTask indexes the status, priority and dates of a task so that
// tasks can be queried.
func transferTask(doc *bluge.Document, block *content.Block) {
	if status := block.TaskStatus(); status != content.TaskStatusNone {
		doc.AddField(bluge.NewKeywordField("task", status.String()))

		state := "open"
		if status.IsClosed() {
			state = "closed"
		}
		doc.AddField(bluge.NewKeywordField("task:state", state))
	}

	if priority := priorityValue(block.Priority()); priority != "" {
		// The values A, B and C sort in the order of the priorities.
		doc.AddField(bluge.NewKeywordField("priority", priority).Sortable())
	}

	if scheduled := block.Scheduled(); scheduled != nil {
		doc.AddField(bluge.NewDateTimeField("scheduled", scheduled.Date))
	}

	if deadline := block.Deadline(); deadline != nil {
		doc.AddField(bluge.NewDateTimeField("deadline", deadline.Date))
	}
}

// addTitleSort adds the title of the page to a document of the page or of a
// block on it, so that results can be sorted by title. The title field itself
// is split into words, which can not be sorted on.
func addTitleSort(doc *bluge.Document, page *Page) {
	if page.Title == "" {
		return
	}

	doc.AddField(bluge.NewKeywordField(SortFieldTitle, normalizeRef(page.Title)).Sortable())
}

// priorityValue is the value a priority is indexed as, or an empty string for
// no priority.
func priorityValue(priority content.Priority) string {
	switch priority {
	case content.PriorityA:
		return "A"
	case content.PriorityB:
		return "B"
	case content.PriorityC:
		return "C"
	}

	return ""
}

// formatLocation writes a location as it is stored in the index, such as
// "0.1".
func formatLocation(location []int) string {
	parts := make([]string, len(location))
	for i, idx := range location {
		parts[i] = strconv.Itoa(idx)
	}
	return strings.Join(parts, ".")
}

// blockIDAt returns the ID that blockID gives the block at the location on the
// page with the sub path, see utils.LocationOf.
func blockIDAt(subPath string, location []int) string {
	var path strings.Builder
	path.WriteString(subPath)
	for i := len(location) - 1; i >= 0; i-- {
		path.WriteRune(':')
		path.WriteString(strconv.Itoa(location[i]))
	}

	// The root of the page is the last part of the ID.
	path.WriteString(":0")
	return path.String()
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
			if _, ok := sibling.(*content.Block); ok {
				idx++
			}
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

// transferProperties indexes properties so that they can be queried. If store
// is set the plain text value of each property is also stored, so that it can
// be returned in search results without opening the page.
func (i *BlugeIndex) transferProperties(doc *bluge.Document, properties *content.Properties, store bool) {
	for _, node := range properties.Children() {
		prop, ok := node.(*content.Property)
		if !ok {
			continue
		}

		// The name is indexed on its own so that a query can find what has a
		// property, also when the property has no value.
		doc.AddField(bluge.NewKeywordField("props", normalizeRef(prop.Name)))

		i.transferRefs(doc, "prop:"+prop.Name, prop.Children())

		s := plainText(prop.Children())
		if s == "" {
			continue
		}

		doc.AddField(bluge.NewTextField("prop:"+prop.Name+":text", s))
		value := bluge.NewKeywordField("prop:"+prop.Name+":value", s)
		if store {
			value.StoreValue()
		}
		doc.AddField(value)
		doc.AddField(bluge.NewKeywordField(SortFieldProperty(prop.Name), sortableValue(s)).Sortable())

		// Numbers and dates are also indexed as such, so that they can be
		// compared.
		if numberPattern.MatchString(s) {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				doc.AddField(bluge.NewNumericField("prop:"+prop.Name+":number", f))
			}
		}

		if date, ok := i.propertyDate(prop, s); ok {
			doc.AddField(bluge.NewDateTimeField("prop:"+prop.Name+":date", date))
		}
	}
}

var numberPattern = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

var journalDayPattern = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)

// propertyDate returns the date that the value of a property is, at midnight
// in the local time zone: a date such as 2024-01-15, a journal day such as
// 20240115, or a reference to a journal such as [[Jan 15th, 2024]]. text is
// the value as plain text.
func (i *BlugeIndex) propertyDate(prop *content.Property, text string) (time.Time, bool) {
	if date, err := time.ParseInLocation("2006-01-02", text, time.Local); err == nil {
		return date, true
	}

	if m := journalDayPattern.FindStringSubmatch(text); m != nil {
		year, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		day, _ := strconv.Atoi(m[3])
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
		if date.Year() == year && int(date.Month()) == month && date.Day() == day {
			return date, true
		}
	}

	refs := prop.Children().PageReferences()
	if i.journalTitles != nil && len(refs) == 1 {
		if date, err := i.journalTitles.Parse(refs[0].(content.PageRef).GetTo()); err == nil {
			return date, true
		}
	}

	return time.Time{}, false
}

// sortableValue returns the value of a property in a form that sorts in the
// correct order as bytes. Numbers sort by their value and before text, and
// text sorts without regard for case.
func sortableValue(value string) string {
	if numberPattern.MatchString(value) {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			// Setting the sign bit of positive numbers and flipping all bits
			// of negative numbers gives bits that sort as the numbers do.
			bits := math.Float64bits(f)
			if bits>>63 == 0 {
				bits |= 1 << 63
			} else {
				bits = ^bits
			}
			return fmt.Sprintf("n%016x", bits)
		}
	}

	return "t" + strings.ToLower(value)
}

// transferRefs indexes the page references in the nodes and the nodes below
// them, as both a link and, for hashtags, a tag.
func (i *BlugeIndex) transferRefs(doc *bluge.Document, field string, nodes content.NodeList) {
	refs := nodes.PageReferences()
	for _, ref := range refs {
		doc.AddField(bluge.NewKeywordField(field+":ref", normalizeRef(ref.(content.PageRef).GetTo())))

		if hashtag, ok := ref.(*content.Hashtag); ok {
			doc.AddField(bluge.NewKeywordField(field+":tag", normalizeRef(hashtag.GetTo())))
		}
	}
}

// normalizeRef brings a page title into the form it is indexed and queried in,
// used for the fields that hold a title rather than text. Logseq does not
// distinguish between page titles that only differ in case, so `[[example]]`
// and `[[Example]]` are references to the same page and have to match the same
// query.
func normalizeRef(title string) string {
	return strings.ToLower(title)
}

// transferLinks indexes the URLs that the nodes and the nodes below them link
// to.
func (i *BlugeIndex) transferLinks(doc *bluge.Document, nodes content.NodeList) {
	links := nodes.FilterDeep(content.IsOfType[content.HasLinkURL]())
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
		case *content.Table:
			return plainText(n.Children())
		case *content.MathBlock:
			return n.Value
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
		case *content.RawText:
			builder.WriteString(n.Value)
		case *content.Hashtag:
			builder.WriteString("#")
			builder.WriteString(n.To)
		case *content.PageLink:
			builder.WriteString(n.To)
		case *content.PageRefText:
			builder.WriteString(n.To)
		case *content.CodeSpan:
			builder.WriteString(n.Value)
		case *content.CodeBlock:
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}

			builder.WriteString(n.Code)
		case *content.Math:
			builder.WriteString(n.Value)
		case *content.MathBlock:
			if builder.Len() > 0 {
				builder.WriteString("\n\n")
			}

			builder.WriteString(n.Value)
		case *content.TableRow:
			plainText0(n.Children(), builder)
			builder.WriteRune('\n')
		case *content.TableCell:
			plainText0(n.Children(), builder)
			builder.WriteRune(' ')
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

		mq := bluge.NewMatchQuery(query.text).SetField(query.field)
		if !query.partial {
			mq = mq.SetOperator(bluge.MatchQueryOperatorAnd)
		}
		return mq
	case *fieldEquals:
		if strings.HasPrefix(query.field, "prop:") {
			return bluge.NewTermQuery(query.value).SetField(query.field + ":value")
		}

		value := query.value
		if query.normalized {
			value = normalizeRef(value)
		}

		return bluge.NewTermQuery(value).SetField(query.field)
	case *fieldPattern:
		return bluge.NewRegexpQuery(query.pattern).SetField(query.field)
	case *numberRange:
		return bluge.NewNumericRangeInclusiveQuery(query.min, query.max, query.minInclusive, query.maxInclusive).SetField(query.field)
	case *dateRange:
		return bluge.NewDateRangeQuery(query.start, query.end).SetField(query.field)
	case *fieldRefs:
		if query.tag {
			return bluge.NewTermQuery(normalizeRef(query.target)).SetField(query.field + ":tag")
		}

		return bluge.NewTermQuery(normalizeRef(query.target)).SetField(query.field + ":ref")
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
		case "tags":
			page.Tags = append(page.Tags, string(value))
		default:
			if name, ok := storedPropertyName(field); ok {
				if page.PropertyValues == nil {
					page.PropertyValues = make(map[string]string)
				}

				page.PropertyValues[name] = string(value)
			}
		}

		return true
	})

	return page
}

// storedPropertyName returns the name of the property that a stored field
// holds the value of.
func storedPropertyName(field string) (string, bool) {
	name, ok := strings.CutPrefix(field, "prop:")
	if !ok {
		return "", false
	}

	return strings.CutSuffix(name, ":value")
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
					continue
				}

				location = append(location, idx)
			}

			// Reverse the location
			for i, j := 0, len(location)-1; i < j; i, j = i+1, j-1 {
				location[i], location[j] = location[j], location[i]
			}

			// Remove the first element which is the root block index
			if len(location) > 0 {
				location = location[1:]
			}

			block.Location = location
		case "page":
			block.PageSubPath = string(value)
		case "id":
			block.ID = string(value)
		case "preview":
			block.Preview = string(value)
		default:
			if name, ok := storedPropertyName(field); ok {
				if block.PropertyValues == nil {
					block.PropertyValues = make(map[string]string)
				}

				block.PropertyValues[name] = string(value)
			}
		}

		return true
	})

	return block
}
