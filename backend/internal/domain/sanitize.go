package domain

import "strings"

// unsafeMarkupCharacter is the set of characters that have no legitimate use
// in the free-text fields of this system (a name, a document number, a
// plate) but do have a well known role in HTML/script injection. Rejecting
// them at the domain boundary means every write path -- API, seed script,
// future importer -- gets the same guarantee, instead of relying on each
// caller to remember to sanitize.
//
// This is a defense-in-depth measure: the repositories already use
// parameterized queries, so none of these characters lead to SQL injection.
// The risk this closes is a stored value later rendered unescaped outside
// the React frontend (a PDF report, an HTML email, a plain log viewer).
const unsafeMarkupCharacter = "<>"

// hasUnsafeMarkup reports whether value contains a character with no
// legitimate use in a workshop record's free-text fields.
func hasUnsafeMarkup(value string) bool {
	return strings.ContainsAny(value, unsafeMarkupCharacter)
}
