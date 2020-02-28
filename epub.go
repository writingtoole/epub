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
// It doesn't matter what order your code calls AddImage, AddXHTML,
// AddJavascript, or AddStylesheet to put files in the ePub book. Nor
// does it matter what order your code calls AddNavpoints to add files
// to the book spine.
//
// ePub files are specially formatted zip archives. You can unzip the
// resulting .epub file and inspect the contents if needed.
//
// Limitations
//
// Currently this package doesn't support adding fonts or JavaScript
// files, nor does it support encrypted or DRM'd books.
//
// By default this package writes out ePub v2.0 format files. You can
// write V3 files either by calling the WriteV3 method directly, or
// setting the epub object's version to v3 by e.SetVersion(3).
package epub

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	img "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// EPub holds the contents of the ePub book.
type EPub struct {
	version   float64
	metadata  []metadata
	images    []image
	xhtml     []xhtml
	navpoints []*Navpoint
	styles    []style
	scripts   []javascript
	fonts     []font
	lastId    map[string]int
	uuid      string
	title     string
	authors   []string
	artists   []string
	// If true then do a bit of preprocessing to xhtml
	// files when writing v3 format books.
	fixV2XHTML bool
	coverID    Id
}

type pair struct {
	key string
	// Key prefix for ePub v2 books
	v2prefix string
	// Key prefix for ePub v3 books
	v3prefix string
	value    string
	// Metadata scheme
	scheme string
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

type javascript struct {
	name     string
	contents string
	id       Id
}

type font struct {
	name     string
	contents []byte
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

// NamespaceUUID is the namespace we're using for all V5 UUIDs
var NamespaceUUID = uuid.Must(uuid.FromString("443ed275-966f-4099-8bee-5a6e1e474bb4"))

// New creates a new empty ePub file.
func New() *EPub {
	ret := &EPub{lastId: make(map[string]int), version: 2, fixV2XHTML: true}
	u, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Sprintf("can't create UUID: %v", err))
	}
	ret.uuid = "urn:uuid:" + u.String()
	ret.metadata = append(ret.metadata, metadata{
		kind:  "dc:identifier",
		value: ret.uuid,
		pairs: []pair{{key: "id", value: "BookId"}},
	})

	return ret
}

// SetVersion sets the default version of the ePub file. Throws an
// error if an unrecognized version is specified; currently only 2 and
// 3 are recognized.
func (e *EPub) SetVersion(version float64) error {
	if version != 2 && version != 3 {
		return fmt.Errorf("EPub version %v is unsupported", version)
	}
	e.version = version
	return nil
}

func (e *EPub) Version() float64 {
	return e.version
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
	for _, m := range e.metadata {
		if m.kind == "dc.identifier" {
			m.value = e.uuid
		}
	}
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

// AddJavaScript adds a JavaScript file to the ePub book. Path is the
// relative path in the book to the javascript file, and contents is
// the JavaScript itself.
//
// Returns the ID of the added file, or an error if something went wrong.
func (e *EPub) AddJavaScript(path, contents string) (Id, error) {
	j := javascript{name: path, contents: contents, id: e.nextId("js")}
	e.scripts = append(e.scripts, j)
	return j.id, nil
}

// AddJavaScriptFile adds the named JavaScript file to the ePub
// book. source is the name of the file to be added while dest is the
// name the file should have in the ePub book.
//
// Returns the ID of the added file, or an error if something went
// wrong reading the file.
func (e *EPub) AddJavaScriptFile(source, dest string) (Id, error) {
	c, err := ioutil.ReadFile(source)
	if err != nil {
		return "", err
	}
	return e.AddJavaScript(dest, string(c))
}

// AddFont adds a font to the ePub book. Path is the relative path in
// the book to the font, and contents is the contents of the font.
//
// Returns the ID of the added file, or an error if something went wrong.
func (e *EPub) AddFont(path string, contents []byte) (Id, error) {
	if !strings.HasSuffix(path, ".otf") {
		return "", errors.New("Only opentype fonts are supported")
	}

	f := font{name: path, contents: contents, id: e.nextId("font")}
	e.fonts = append(e.fonts, f)
	return f.id, nil
}

// AddFontFile adds the named font to the epub book. Source is the
// name of the file to be added while dest is the name the file should
// have in the ePub book.
//
// Returns the ID of the added file, or an error if something went wrong.
func (e *EPub) AddFontFile(source, dest string) (Id, error) {
	c, err := ioutil.ReadFile(source)
	if err != nil {
		return "", err
	}
	return e.AddFont(dest, c)
}

// AddXHTML adds an xhtml file to the ePub book. Path is the relative
// path to this file in the book, and contents is the contents of the
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
			{key: "name", value: "cover"},
			{key: "content", value: string(id)},
		},
	}
	e.metadata = append(e.metadata, m)
	e.coverID = id
}

// Write out the book to the named file. The book will be written
// in whichever version the epub object is tagged with. By default
// this is V2.
func (e *EPub) Write(name string) error {
	log.Printf("Writing version %v", e.version)
	switch e.version {
	case 2:
		return e.WriteV2(name)
	case 3:
		return e.WriteV3(name)
	default:
		return fmt.Errorf("Unable to write epub version %v files", e.version)
	}
}
