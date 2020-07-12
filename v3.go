package epub

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"sort"
	"time"
)

// This file holds the code to write epub version 3 format files.

// sub performs a global search and replace, because Go's regex
// package is a bit inconvenient to use.
func sub(in, regex, repl string) string {
	r := regexp.MustCompile(regex)
	return r.ReplaceAllString(in, repl)
}

func (e *EPub) WriteV3(name string) error {
	buf, err := e.SerializeV3()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(name, buf, 0666); err != nil {
		return err
	}

	return nil
}

func (e *EPub) SerializeV3() ([]byte, error) {
	buf := new(bytes.Buffer)
	z := zip.NewWriter(buf)

	// Make sure we're using deflate, which is the only compression
	// scheme that ePub officially suports. This is the default, but we
	// do this to be extra careful. Since we're registering a compressor
	// anyway we also turn on max compression. This doesn't make much
	// difference for most books (text compresses really well already,
	// and images don't) but that's fine.
	z.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})

	// add mimetype
	w, err := z.Create("mimetype")
	if err != nil {
		return nil, err
	}
	fmt.Fprint(w, "application/epub+zip")

	// Add the images.
	for _, i := range e.images {
		w, err = z.Create("OPS/" + i.name)
		if err != nil {
			return nil, err
		}
		length, err := w.Write(i.contents)
		if err != nil {
			return nil, fmt.Errorf("unable to write %v, %v of %v bytes: %v", i.name, length, len(i.contents), err)
		}
	}

	// Add the xhtml.
	for _, x := range e.xhtml {
		w, err = z.Create("OPS/" + x.name)
		if err != nil {
			return nil, err
		}
		c := x.contents
		if e.fixV2XHTML {
			c = fixV2XHTML(c)
		}
		length, err := w.Write([]byte(c))
		if err != nil {
			return nil, fmt.Errorf("unable to write %v, %v of %v bytes: %v", x.name, length, len(x.contents), err)
		}
	}

	// Add the css.
	for _, s := range e.styles {
		w, err = z.Create("OPS/" + s.name)
		if err != nil {
			return nil, err
		}
		length, err := w.Write([]byte(s.contents))
		if err != nil {
			return nil, fmt.Errorf("unable to write %v, %v of %v bytes: %v", s.name, length, len(s.contents), err)
		}
	}

	// Add the javascript.
	for _, s := range e.scripts {
		w, err = z.Create("OPS/" + s.name)
		if err != nil {
			return nil, err
		}
		length, err := w.Write([]byte(s.contents))
		if err != nil {
			return nil, fmt.Errorf("unable to write %v, %v of %v bytes: %v", s.name, length, len(s.contents), err)
		}
	}

	// Add the fonts.
	for _, f := range e.fonts {
		w, err = z.Create("OPS/" + f.name)
		if err != nil {
			return nil, err
		}
		length, err := w.Write(f.contents)
		if err != nil {
			return nil, fmt.Errorf("unable to write %v, %v of %v bytes: %v", f.name, length, len(f.contents), err)
		}
	}

	if err = e.addTocV3(z); err != nil {
		return nil, err
	}

	if err = e.addContainerV3(z); err != nil {
		return nil, err
	}

	if err = e.addRenditionsV3(z); err != nil {
		return nil, err
	}

	// Done adding stuff. Close off the file and write it out.
	if err = z.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil

}

func (e *EPub) obfuscate(raw []byte) []byte {

	return nil
}

func (e *EPub) addContainerV3(z *zip.Writer) error {
	w, err := z.Create("META-INF/container.xml")
	if err != nil {
		return err
	}
	fmt.Fprintf(w, `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
`)
	for _, fn := range e.renditionNamesV3() {
		fmt.Fprintf(w, "    <rootfile full-path=\"OPS/%s\" media-type=\"application/oebps-package+xml\" />\n", fn)
	}
	fmt.Fprintf(w, `  </rootfiles>
</container>
`)

	return nil
}

// renditionNamesV3 returns the base filenames for the different
// renditions in the book. For the moment there's only one, but at
// some point hopefully we'll add the ability to have alternates.
func (e *EPub) renditionNamesV3() []string {
	return []string{"book.opf"}
}

// addRenditionsV3 adds the different .opf rendition files to the
// epub. At the moment this means the single book.opf file.
func (e *EPub) addRenditionsV3(z *zip.Writer) error {

	w, err := z.Create("OPS/book.opf")
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	fmt.Fprintf(w, "<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"BookId\">\n")

	e.addV3Metadata(w)
	e.addV3Manifest(w)
	e.addV3Spine(w)

	fmt.Fprintf(w, "</package>\n")

	return nil
}

func (e *EPub) addV3Metadata(w io.Writer) error {
	fmt.Fprintf(w, "  <metadata xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n")
	idCount := 0
	seenDCTerms := false
	for _, m := range e.metadata {
		idCount++
		switch m.kind {
		case "meta":
			// We skip the meta entries, they're probably cover image
		case "dc:identifier":
			fmt.Fprintf(w, `    <dc:identifier id="BookId">%s</dc:identifier>`, m.value)
			fmt.Fprintf(w, "\n")
		default:
			// Note if we've seen a modified time entry. We need one, and
			// will add one if necessary.
			if m.kind == "dcterms:modified" {
				seenDCTerms = true
			}
			fmt.Fprintf(w, `    <%s id="id%v"`, m.kind, idCount)
			// If there's a value then it's a container-style XML thing
			if len(m.value) != 0 {
				fmt.Fprintf(w, ">%s</%s>\n", m.value, m.kind)
			} else {
				// No value means plain standalone element XML thing
				fmt.Fprintf(w, " />\n")
			}
			// Write out the modifiers.
			for _, p := range m.pairs {
				fmt.Fprintf(w, `    <meta refines="#id%v" property="%s%s"`, idCount, p.v3prefix, p.key)
				if p.scheme != "" {
					fmt.Fprintf(w, ` scheme="%s"`, p.scheme)
				}
				fmt.Fprintf(w, ">%s</meta>\n", p.value)
			}
		}
	}
	if !seenDCTerms {
		fmt.Fprintf(w, "    <meta property=\"dcterms:modified\">%s</meta>\n", time.Now().Format("2006-01-02T15:04:05Z"))
	}
	if e.seriesName != "" || e.setName != "" {
		if e.seriesName != "" {
			fmt.Fprintf(w, "    <meta property=\"belongs-to-collection\" id=\"seriesinfo\">%s</meta>\n", e.seriesName)
			fmt.Fprint(w, "    <meta refines=\"#seriesinfo\" property=\"collection-type\">series</meta>\n")
		}
		if e.setName != "" {
			fmt.Fprintf(w, "    <meta property=\"belongs-to-collection\" id=\"seriesinfo\">%s</meta>\n", e.setName)
			fmt.Fprint(w, "    <meta refines=\"#seriesinfo\" property=\"collection-type\">set</meta>\n")
		}
		if e.entry != "" {
			fmt.Fprintf(w, "    <meta refines=\"#seriesinfo\" property=\"group-position\">%s</meta>\n", e.entry)
		}
	}
	fmt.Fprintf(w, "  </metadata>\n")

	return nil
}

func (e *EPub) addV3Manifest(w io.Writer) error {
	fmt.Fprintf(w, "  <manifest>\n")

	for _, i := range e.images {
		extraBits := ""
		if i.id == e.coverID {
			extraBits += ` properties="cover-image"`
		}
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q %s/>\n", i.id, i.name, "image/"+i.filetype, extraBits)
	}
	for _, x := range e.xhtml {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", x.id, x.name, "application/xhtml+xml")
	}
	for _, s := range e.styles {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", s.id, s.name, "text/css")
	}
	for _, s := range e.scripts {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", s.id, s.name, "application/javascript")
	}
	for _, f := range e.fonts {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", f.id, f.name, "application/opentype")
	}
	// Add an entry for our TOC. Needs the "nav" property to note TOC-ness.
	fmt.Fprintf(w, "    <item id=%q properties=%q href=%q media-type=%q	/>\n", "ncx", "nav", "__toc.xhtml", "application/xhtml+xml")
	fmt.Fprintf(w, "  </manifest>\n")
	return nil
}

func (e *EPub) addV3Spine(w io.Writer) error {
	fmt.Fprintf(w, "  <spine>\n")
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

// fixV2XHTML patches up epub v2-compatible xhtml to make it v3
// compatible. It's annoying to have to do this, but files that are
// fine for v2 don't work for v3, and vice versa.
func fixV2XHTML(o string) string {
	ret := o
	// v2 xhtml wants a:
	// <!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
	// tag, but v3 wants:
	// <!DOCTYPE html>
	// so strip out the extra doctype bits for v3 cleanup.
	ret = sub(ret, `^(?ms)(<\?xml[^>]*>\s*<!DOCTYPE)\b[^>]*>`, "$1 html>")

	return ret
}

func (e *EPub) addTocV3(z *zip.Writer) error {
	w, err := z.Create("OPS/__toc.xhtml")
	if err != nil {
		return err
	}

	fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE xhtml>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head>
<title>%s</title>
</head>
<body>`, e.title)
	fmt.Fprintf(w, `<nav epub:type="toc" id="toc">
  <h1>Table of Contents</h1>
`)
	writeV3Navpoints(e.navpoints, "    ", w)

	fmt.Fprintf(w, "</nav>\n")
	fmt.Fprintf(w, "</body>\n")
	fmt.Fprintf(w, "</html>\n")
	return nil
}

func writeV3Navpoints(np []*Navpoint, prefix string, w io.Writer) {
	fmt.Fprintf(w, "%s<ol>\n", prefix)
	sort.Slice(np, func(i, j int) bool { return np[i].order < np[j].order })

	for _, n := range np {
		fmt.Fprintf(w, "%s  <li>\n", prefix)
		fmt.Fprintf(w, "%s    <a href=%q>%s</a>\n", prefix, n.filename, n.label)

		if len(n.navpoints) != 0 {
			writeV3Navpoints(n.navpoints, prefix+"  ", w)
		}
		fmt.Fprintf(w, "%s</li>\n", prefix)
	}

	fmt.Fprintf(w, "%s</ol>\n", prefix)
}
