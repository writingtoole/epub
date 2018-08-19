package epub_test

import (
	"fmt"

	"github.com/writingtoole/epub"
)

func ExampleNew() {
	b := epub.New()
	b.AddAuthor("me")
	b.AddLanguage("en")
	b.SetTitle("My Book")

	// Put our cover into the file, with a path in the book of
	// "images/cover.jpg".
	coverId, _ := b.AddImageFile("source/mycover.jpg", "images/cover.jpg")
	b.SetCoverImage(coverId)

	// Add our pre-formatted cover file. The path in the book will be
	// "xhtml/cover.xhtml". It's noted as file 0 in the book since we'll
	// be adding files in a different order than we want them to appear.
	//
	// We don't add the cover to the book's TOC.
	b.AddXHTMLFile("source/cover_file", "xhtml/cover.xhtml", 1)

	// Add chapter 1. We start numbering chapter files with 10.
	b.AddXHTMLFile("source/file1.xhtml", "xhtml/file1.xhtml", 10)
	np1 := b.AddNavpoint("Chapter 1", "xhtml/file1.xhtml", 10)
	// Chapter 1 has 10 fragments that each get a spine entry.
	for i := 1; i <= 10; i++ {
		// Add these section fragments to the spine.
		np1.AddNavpoint(fmt.Sprintf("Section %v", i), fmt.Sprintf("xhtml/file1.xhtml#%v", i), i)
	}

	// Add chapter 2.
	b.AddXHTMLFile("source/file2.xhtml", "xhtml/file2.xhtml", 11)
	b.AddNavpoint("Chapter 2", "xhtml/file2.xhtml", 11)

	// Add an actual table of contents. This appears inline in the book,
	// an dis is different than (and unrelated to) the one built from
	// navpoints.
	b.AddXHTMLFile("source/toc.xhtml", "xhtml/toc.xhtml", 2)

	b.Write("mybook.epub")
}
