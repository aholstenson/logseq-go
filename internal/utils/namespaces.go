package utils

import "strings"

// NamespaceOf returns the namespace that a page title belongs to, which is the
// part of the title before its last `/`. Titles that are not part of a
// namespace return an empty string.
func NamespaceOf(title string) string {
	parts := namespaceParts(title)
	if len(parts) < 2 {
		return ""
	}

	return strings.Join(parts[:len(parts)-1], "/")
}

// NamespacesOf returns all the namespaces that a page title is part of, from
// the one it belongs to up to the top one. A title that is not part of a
// namespace returns nothing.
func NamespacesOf(title string) []string {
	parts := namespaceParts(title)
	if len(parts) < 2 {
		return nil
	}

	namespaces := make([]string, 0, len(parts)-1)
	for i := len(parts) - 1; i > 0; i-- {
		namespaces = append(namespaces, strings.Join(parts[:i], "/"))
	}

	return namespaces
}

// namespaceParts splits a title into the parts its namespaces are made up of.
// Empty parts are dropped, so that a stray `/` does not turn into a namespace
// without a name.
func namespaceParts(title string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(title, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}
