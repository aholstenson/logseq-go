//go:build race

package markdown_test

// timeLimitFactor is how much longer the time limits of the parse time tests
// are. The race detector makes the code run many times slower.
const timeLimitFactor = 20
