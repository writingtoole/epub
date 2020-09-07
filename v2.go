package epub

// This file holds the code to write epub version 2 format files.

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strconv"
)

// Write emits an epub V2 format the epub to the named file.
func (e *EPub) WriteV2(name string) error {
	buf, err := e.SerializeV2()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(name, buf, 0666); err != nil {
		return err
	}

	return nil

}

// SerializeV2 returns a byteslice containing the built epub.
func (e *EPub) SerializeV2() ([]byte, error) {
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

	// add mimetype. Need to use the CreateHeader method because the
	// mimetype file needs to be uncompressed.
	h := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	w, err := z.CreateHeader(h)
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
		length, err := w.Write([]byte(x.contents))
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

	if err = e.addContent(z); err != nil {
		return nil, err
	}

	if err = e.addToc(z); err != nil {
		return nil, err
	}

	if err = e.addContainer(z); err != nil {
		return nil, err
	}

	if err = z.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil

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
	for _, s := range e.scripts {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", s.id, s.name, "application/javascript")
	}
	for _, f := range e.fonts {
		fmt.Fprintf(w, "    <item id=%q href=%q media-type=%q />\n", f.id, f.name, "application/opentype")
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
			fmt.Fprintf(w, ` %s%s="%s"`, p.v2prefix, p.key, p.value)
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
