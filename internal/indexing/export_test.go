package indexing

// NewBlugeIndexWithVersion opens an index as if the index version was the
// given one, so that tests can open an index that was built with another
// version.
var NewBlugeIndexWithVersion = newBlugeIndex

// IndexVersion is the current version of the index.
const IndexVersion = indexVersion
