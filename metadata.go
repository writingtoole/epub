package epub

// AddLanguage adds a language for the book. This should be an RFC3066
// language code.
func (e *EPub) AddLanguage(l string) {
	e.addDcItem("language", l)
}

// SetTitle sets the title of the book.
func (e *EPub) SetTitle(title string) {
	e.title = title
	e.addDcItem("title", title)
}

// AddAuthor adds an author's name to the list of authors for the book.
func (e *EPub) AddAuthor(author string) {
	e.authors = append(e.authors, author)
	m := metadata{
		kind:  "dc:creator",
		value: author,
		pairs: []pair{{key: "opf:role", value: "aut"}},
	}
	e.metadata = append(e.metadata, m)
}

// AddPublisher adds a publisher entry for the book.
func (e *EPub) AddPublisher(pub string) {
	e.addDcItem("publisher", pub)
}

// AddDescripton adds a description entry for the book.
func (e *EPub) AddDescription(desc string) {
	e.addDcItem("description", desc)
}

// AddSubject adds a subject entry for the book.
func (e *EPub) AddSubject(subj string) {
	e.addDcItem("subject", subj)
}

func (e *EPub) addDcItem(i, v string) {
	m := metadata{kind: "dc:" + i, value: v}

	e.metadata = append(e.metadata, m)
}
