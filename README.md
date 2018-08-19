# epub

A Go library for the creation of ePub ebook files.

This package creates basic ePub v2.0 format files, suitable for
creating books.

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

It doesn't matter what order your code calls AddImage, AddXHTML, or
AddStylesheet to put files in the ePub book. Nor does it matter
what order your code calls AddNavpoints to add files to the
book spine.

ePub files are specially formatted zip archives. You can unzip the
resulting .epub file and inspect the contents if needed.

# Limitations

Currently this package doesn't support adding fonts or JavaScript
files, nor does it support encrypted or DRM'd books.

This package intentionally writes out ePub v2.0 format files. The
current standard version is (as of 8/2018) v3.1. All ePub readers
can manage v2.0 files but not all can manage 3.x, which is why
we're writing the older format.

# License

epub is made available under the terms of the [New BSD License](http://opensource.org/licenses/BSD-3-Clause)