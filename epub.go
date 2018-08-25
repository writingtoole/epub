// Package epub creates ePub v2.0 format ebooks.
//
// An ePub file consists of one or more XHTML files that represent
// the text of your book, the resources those files reference, and the
// optional structured metadata (such as author and publisher) for the
// book.
//
// This library doesn't do validity testing of the book file, so it's
// possible to create invalid books. Testing the output with external
// ePub validators such as ePubCheck
// (https://github.com/IDPF/epubcheck) is advisable.
//
// Structure notes
//
// All files in an ePub should be reachable, directly or indirectly,
// from the spine of the book. Books with unreferenced files are
// technically illegally formatted.
//
// It doesn't matter what order your code calls AddImage, AddXHTML, or
// AddStylesheet to put files in the ePub book. Nor does it matter
// what order your code calls AddNavpoints to add files to the
// book spine.
//
// ePub files are specially formatted zip archives. You can unzip the
// resulting .epub file and inspect the contents if needed.
//
// Limitations
//
// Currently this package doesn't support adding fonts or JavaScript
// files, nor does it support encrypted or DRM'd books.
//
// This package intentionally writes out ePub v2.0 format files. The
// current standard version is (as of 8/2018) v3.1. All ePub readers
// can manage v2.0 files but not all can manage 3.x, which is why
// we're writing the older format.
package epub

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"github.com/satori/go.uuid"

	img "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// EPub holds the contents of the ePub book.
type EPub struct {
	metadata  []metadata
	images    []image
	xhtml     []xhtml
	navpoints []*Navpoint
	styles    []style
	lastId    map[string]int
	uuid      string
	title     string
	authors   []string
	artists   []string
}

type pair struct {
	key   string
	value string
}

type metadata struct {
	kind  string
	value string
	pairs []pair
}

type style struct {
	name     string
	contents string
	id       Id
}

type xhtml struct {
	name      string
	contents  string
	id        Id
	order     int // Explicit ordering for file
	baseOrder int // Implicit order for file
}

type image struct {
	name     string
	contents []byte
	filetype string
	id       Id
}

// Id holds an identifier for an item that's been added to the book.
type Id string

// Navpoint represents an entry in the book's spine.
type Navpoint struct {
	label     string
	filename  string
	order     int
	navpoints []*Navpoint
}

// New creates a new empty ePub file.
func New() *EPub {
	ret := &EPub{lastId: make(map[string]int)}
	u, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("can't create UUID: %v", err))
	}
	ret.uuid = "urn:uuid:" + u.String()
	ret.metadata = append(ret.metadata, metadata{
		kind:  "dc:identifier",
		value: ret.uuid,
		pairs: []pair{{"id", "BookId"}},
	})

	return ret
}

// UUID returns the currently assigned UUID for this epub.
func (e *EPub) UUID() string {
	return strings.TrimPrefix("urn:uuid:", e.uuid)
}

// SetUUID overrides the default UUID assigned to this epub. Since
// many ebook readers use the UUID to identify a book it's usually
// wise to assign the same UUID to different revisions of a book.
func (e *EPub) SetUUID(uu string) error {
	u, err := uuid.FromString(uu)
	if err != nil {
		return err
	}
	e.uuid = "urn:uuid:" + u.String()
	return nil
}

func (e *EPub) nextId(class string) Id {
	last, ok := e.lastId[class]
	if !ok {
	}
	last++
	e.lastId[class] = last
	return Id(class + strconv.Itoa(last))
}

// AddImage adds an image to the ePub book. Path is the relative path
// in the book to the image, and contents is the image itself.
//
// The library will autodetect the filetype for the image from the
// contents. Some ebook reading software infers filetype from the
// filename, so while it isn't required it is prudent to have the file
// extension match the filetype.
func (e *EPub) AddImage(path string, contents []byte) (Id, error) {
	_, fmt, err := img.DecodeConfig(bytes.NewReader(contents))
	if err != nil {
		return "", err
	}

	i := image{name: path, filetype: fmt, contents: contents, id: e.nextId("img")}

	e.images = append(e.images, i)
	return i.id, nil
}

// AddImageFile adds an image file to the ePub book. source is the
// name of the file to be added while dest is the name the file should have
// in the ePub book.
//
// Returns the ID of the added file, or an error if something went
// wrong reading the file.
func (e *EPub) AddImageFile(source, dest string) (Id, error) {
	c, err := ioutil.ReadFile(source)
	if err != nil {
		return "", err
	}
	return e.AddImage(dest, c)
}

// AddXHTML adds an xhtml file to the ePub book. Path is the relative
// path to this fie in the book, and contents is the contents of the
// xhtml file.
//
// By default each file appears in the book's spine in the order they
// were added. You may, if you wish, optionally specify the
// ordering. (Note that all files without an order specified get an
// implicit order of '0') If multiple files are given the same order
// then they're sub-sorted by the order they were added.
func (e *EPub) AddXHTML(path string, contents string, order ...int) (Id, error) {
	if len(order) > 1 {
		return "", fmt.Errorf("Too many order parameters given")
	}
	o := 0
	if len(order) == 1 {
		o = order[0]
	}
	x := xhtml{
		name:      path,
		contents:  contents,
		id:        e.nextId("xhtml"),
		order:     o,
		baseOrder: len(e.xhtml),
	}
	e.xhtml = append(e.xhtml, x)
	return x.id, nil
}

// AddXHTMLFile adds an xhtml file currently on-disk to the ePub
// book. source is the name of the file to add, while dest is the name
// the file should have in the ePub book.
//
// Returns the ID of the added file, or an error if something went
// wrong.
func (e *EPub) AddXHTMLFile(source, dest string, order ...int) (Id, error) {
	c, err := ioutil.ReadFile(source)
	if err != nil {
		return "", err
	}
	return e.AddXHTML(dest, string(c), order...)
}

// AddNavpoint adds a top-level navpoint.
//
// Navpoints are part of the book's table of contents. The label is
// the string that will be shown in the TOC (note that many ereaders
// do *not* do HTML unescaping). Name is the URI of the point in the
// book this navpoint points to. Not every file in a book needs a
// navpoint that points to it -- all navpoints are optional.
//
// Some ereaders do not permit fragment IDs in the URI for top-level navpoints.
//
// The order parameter is used to sort the navpoints when building the
// book's TOC.  Navpoints do not need to be added to the book in the
// order they appear in the TOC, the order numbers do not have to
// start from 1, and there may be gaps in the order.
//
// Note that the order that entries appear in the table of contents,
// and the order that files appear in the book, don't have to be
// related.
//
// Also note that some ereader software will elide entries from the
// book's TOC. (iBooks 1.15 on OS X, for example, won't display
// entries labeled "Cover" or "Table of Contents")
func (e *EPub) AddNavpoint(label string, name string, order int) *Navpoint {
	n := &Navpoint{label: label, filename: name, order: order}
	e.navpoints = append(e.navpoints, n)
	return n
}

// AddNavpoint adds a child navpoint. Label is the name that will be
// shown in the TOC, name is the URI of the point in the book this
// navpoint points to, and order is the order of the navpoint in the
// TOC.
//
// Child URIs typically refer to a point in the parent Navpoint's file
// and, indeed, some ereaders require this. That is, if the parent
// navpoint has a file of "foo/bar.xhtml" the child navpoints must be
// fragments inside that file (such as "foo/bar.xhtml#Point3").
func (n *Navpoint) AddNavpoint(label string, name string, order int) *Navpoint {
	nn := &Navpoint{label: label, filename: name, order: order}
	n.navpoints = append(n.navpoints, nn)
	return nn
}

// AddStylesheet adds a CSS stylesheet to the ePub book. Path is the
// relative path to the CSS file in the book, while contents is the
// contents of the stylesheet.
func (e *EPub) AddStylesheet(path, contents string) (Id, error) {
	s := style{name: path, contents: contents, id: e.nextId("css")}
	e.styles = append(e.styles, s)
	return s.id, nil
}

// AddStylesheetFile adds the named file to the ePub as a CSS
// stylesheet. source is the name of the file on disk, while dest is
// the name the stylesheet has in the ePub file.
func (e *EPub) AddStylesheetFile(source, dest string) (Id, error) {
	c, err := ioutil.ReadFile(source)
	if err != nil {
		return "", err
	}
	return e.AddStylesheet(dest, string(c))

}

// SetCoverImage notes which image is the cover.
//
// ePub readers will generally use this as the image displayed in the
// bookshelf. They generally will not display this image when the book
// is read; if you want the first page of your book to have this cover
// image it's best to generate an XHTML file that references the image
// and set it to be the first entry in your spine.
func (e *EPub) SetCoverImage(id Id) {
	m := metadata{
		kind: "meta",
		pairs: []pair{
			{"name", "cover"},
			{"content", string(id)},
		},
	}
	e.metadata = append(e.metadata, m)
}

// Write writes out the epub to the named file.
func (e *EPub) Write(name string) error {
	buf := new(bytes.Buffer)
	z := zip.NewWriter(buf)

	// add mimetype
	w, err := z.Create("mimetype")
	if err != nil {
		return err
	}
	fmt.Fprint(w, "application/epub+zip")

	// Add the images.
	for _, i := range e.images {
		w, err = z.Create("OPS/" + i.name)
		if err != nil {
			return err
		}
		length, err := w.Write(i.contents)
		if err != nil {
			return fmt.Errorf("unable to write %v, %v of %v bytes: %v", i.name, length, len(i.contents), err)
		}
	}

	// Add the xhtml.
	for _, x := range e.xhtml {
		w, err = z.Create("OPS/" + x.name)
		if err != nil {
			return err
		}
		length, err := w.Write([]byte(x.contents))
		if err != nil {
			return fmt.Errorf("unable to write %v, %v of %v bytes: %v", x.name, length, len(x.contents), err)
		}
	}

	// Add the css.
	for _, s := range e.styles {
		w, err = z.Create("OPS/" + s.name)
		if err != nil {
			return err
		}
		length, err := w.Write([]byte(s.contents))
		if err != nil {
			return fmt.Errorf("unable to write %v, %v of %v bytes: %v", s.name, length, len(s.contents), err)
		}
	}

	if err = e.addContent(z); err != nil {
		return err
	}

	if err = e.addToc(z); err != nil {
		return err
	}

	if err = e.addContainer(z); err != nil {
		return err
	}

	if err = z.Close(); err != nil {
		return err
	}

	if err = ioutil.WriteFile(name, buf.Bytes(), 0666); err != nil {
		return err
	}

	return nil
}

// addContent adds the content.opf file to the book.
func (e *EPub) addContent(z *zip.Writer) error {
	w, err := z.Create("OPS/content.opf")
	if err != nil {
		return err
	}

	// First the header
	fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookId">
`)

	e.addMetadata(w)
	e.addManifest(w)
	e.addSpine(w)

	// Close it off
	fmt.Fprintf(w, "</package>\n")
	return nil
}

func (e *EPub) addManifest(w io.Writer) error {
	fmt.Fprintf(w, "  <manifest>\n")

	fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", "ncx", "toc.ncx", "application/x-dtbncx+xml")

	for _, i := range e.images {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", i.id, i.name, "image/"+i.filetype)
	}
	for _, x := range e.xhtml {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", x.id, x.name, "application/xhtml+xml")
	}
	for _, s := range e.styles {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", s.id, s.name, "text/css")
	}

	fmt.Fprintf(w, "  </manifest>\n")
	return nil
}

func (e *EPub) addSpine(w io.Writer) error {
	fmt.Fprintf(w, "  <spine toc=\"ncx\">\n")
	x := e.xhtml
	sort.Slice(x, func(i, j int) bool {
		return x[i].order < x[j].order || (x[i].order == x[j].order && x[i].baseOrder < x[j].baseOrder)
	})
	for _, n := range x {
		fmt.Fprintf(w, "    <itemref idref=%q />\n", n.id)
	}
	fmt.Fprintf(w, "  </spine>\n")

	return nil
}

// addMetadata adds the metadata section.
func (e *EPub) addMetadata(w io.Writer) error {
	fmt.Fprintf(w, `  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
`)

	for _, m := range e.metadata {
		fmt.Fprintf(w, `    <%s`, m.kind)
		for _, p := range m.pairs {
			fmt.Fprintf(w, ` %s="%s"`, p.key, p.value)
		}
		// If there's a value then it's a container-style XML thing
		if len(m.value) != 0 {
			fmt.Fprintf(w, ">%s</%s>\n", m.value, m.kind)
		} else {
			// No value means plain standalone element XML thing
			fmt.Fprintf(w, " />\n")
		}
	}

	fmt.Fprintf(w, "  </metadata>\n")
	return nil
}

// addToc adds the toc.ncx file.
func (e *EPub) addToc(z *zip.Writer) error {
	w, err := z.Create("OPS/toc.ncx")
	if err != nil {
		return err
	}

	fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE ncx PUBLIC "-//NISO//DTD ncx 2005-1//EN" "http://www.daisy.org/z3986/2005/ncx-2005-1.dtd">

<ncx version="2005-1" xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <head>
    <meta name="dtb:uid" content=%q />
    <meta name="dtb:depth" content="1" />
    <meta name="dtb:totalPageCount" content="0" />
    <meta name="dtb:maxPageNumber" content="0" />
  </head>
 `, e.uuid)
	fmt.Fprintf(w, `  <docTitle>
    <text>%s</text>
  </docTitle>
`, e.title)

	if len(e.authors) > 0 {
		fmt.Fprintf(w, "  <docAuthor>\n")
		for _, a := range e.authors {
			fmt.Fprintf(w, "    <text>%s</text>\n", a)
		}
		fmt.Fprintf(w, "  </docAuthor>\n")
	}

	fmt.Fprintf(w, "  <navMap>\n")
	writeNavpoints(e.navpoints, 1, "navpointid", "    ", w)

	fmt.Fprintf(w, "  </navMap>\n")

	fmt.Fprintf(w, "</ncx>\n")
	return nil
}

// addContainer adds the container file to the EPub.
func (e *EPub) addContainer(z *zip.Writer) error {
	w, err := z.Create("META-INF/container.xml")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8"?>

<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles>
    <rootfile full-path="OPS/content.opf" media-type="application/oebps-package+xml" />
  </rootfiles>
</container>`)
	return nil
}

func writeNavpoints(np []*Navpoint, order int, baseID, prefix string, w io.Writer) int {
	sort.Slice(np, func(i, j int) bool { return np[i].order < np[j].order })

	for i, n := range np {
		id := baseID + "_" + strconv.Itoa(i)
		fmt.Fprintf(w, "%s<navPoint id=%q playOrder=\"%v\">\n", prefix, id, order)
		order++
		fmt.Fprintf(w, "%s  <navLabel>\n", prefix)
		fmt.Fprintf(w, "%s    <text>%s</text>\n", prefix, n.label)
		fmt.Fprintf(w, "%s  </navLabel>\n", prefix)
		fmt.Fprintf(w, "%s  <content src=%q />\n", prefix, n.filename)
		if len(n.navpoints) != 0 {
			order = writeNavpoints(n.navpoints, order, id, prefix+"  ", w)
		}
		fmt.Fprintf(w, "%s</navPoint>\n", prefix)
	}
	return order
}
