# epub

A Go library for the creation of ePub ebook files.

This package creates basic ePub v2.0 or V3.0 format files, suitable
for creating books. By default books are tagged and written out as V2,
but you can write V3 format books either by calling the WriteV3 method
directly or by setting the ePub version to 3 via SetVersion(3) and
then calling Write().

An ePub file consists of one or more XHTML files that represent
the text of your book, the resources those files reference, and the
optional structured metadata (such as author and publisher) for the
book.

This library doesn't do validity testing of the book file, so it's
possible to create invalid books. Testing the output with external
ePub validators such as ePubCheck
(https://github.com/IDPF/epubcheck) is advisable.

# Structure notes

All files in an ePub should be reachable, directly or indirectly,
from the spine of the book. Books with unreferenced files are
technically illegally formatted.

It doesn't matter what order your code calls AddImage, AddXHTML,
AddFont, AddJavaScript, or AddStylesheet to put files in the ePub
book. Nor does it matter what order your code calls AddNavpoints to
add files to the book spine.

ePub files are specially formatted zip archives. You can unzip the
resulting .epub file and inspect the contents if needed.

# Limitations

Currently this package doesn't support encrypted or DRM'd books or content.

Fonts are not obscured when writing V3 format files.

None of the interesting bits of the V3 format are currently supported;
v3 books are basically identical to v2 books only using the updated
metadata file formats.

# License

epub is made available under the terms of the [New BSD License](http://opensource.org/licenses/BSD-3-Clause)