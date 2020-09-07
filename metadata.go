package epub

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// AddLanguage adds a language for the book. This should be an RFC3066
// language code.
func (e *EPub) AddLanguage(l string) error {
	e.addDcItem("language", l)
	// Currently we don't validate language codes, though we should.
	return nil
}

// SetTitle sets the title of the book.
func (e *EPub) SetTitle(title string) {
	e.title = title
	e.addDcItem("title", title)
}

// AddAuthor adds an author's name to the list of authors for the book.
func (e *EPub) AddAuthor(author string) {
	e.authors = append(e.authors, author)
	e.AddCreator(author, "aut")
}

func (e *EPub) AddArtist(artist string) {
	e.artists = append(e.artists, artist)
	e.AddCreator(artist, "art")
}

// AddCreator adds a creator entry to the epub file. The creator type
// must come from the list of valid creators at
// http://www.loc.gov/marc/relators/relaterm.html and will return an
// error if an invalid entry is passed.
func (e *EPub) AddCreator(creator string, role string) error {
	if !validRoles[role] {
		return fmt.Errorf("invalid role %v", role)
	}
	m := metadata{
		kind:  "dc:creator",
		value: creator,
		pairs: []pair{{v2prefix: "opf:", key: "role", value: role, scheme: "marc:relators"}},
	}
	e.metadata = append(e.metadata, m)
	return nil
}

// AddContributor adds a creator entry to the epub file. The contributor type
// must come from the list of valid roles at
// http://www.loc.gov/marc/relators/relaterm.html and will return an
// error if an invalid entry is passed.
func (e *EPub) AddContributor(creator string, role string) error {
	if !validRoles[role] {
		return fmt.Errorf("invalid role %v", role)
	}
	m := metadata{
		kind:  "dc:contributor",
		value: creator,
		pairs: []pair{{v2prefix: "opf:", key: "role", value: role, scheme: "marc:relators"}},
	}
	e.metadata = append(e.metadata, m)
	return nil
}

// List of valid roles, from
// http://www.loc.gov/marc/relators/relaterm.html
var validRoles = map[string]bool{
	"abr": true, "act": true, "adp": true, "rcp": true,
	"anl": true, "anm": true, "ann": true, "apl": true, "ape": true,
	"app": true, "arc": true, "arr": true, "acp": true, "adi": true,
	"art": true, "ard": true, "asg": true, "asn": true, "att": true,
	"auc": true, "aut": true, "aqt": true, "aft": true, "aud": true,
	"aui": true, "ato": true, "ant": true, "bnd": true, "bdd": true,
	"blw": true, "bkd": true, "bkp": true, "bjd": true, "bpd": true,
	"bsl": true, "brl": true, "brd": true, "cll": true, "ctg": true,
	"cas": true, "cns": true, "chr": true, "cng": true, "cli": true,
	"cor": true, "col": true, "clt": true, "clr": true, "cmm": true,
	"cwt": true, "com": true, "cpl": true, "cpt": true, "cpe": true,
	"cmp": true, "cmt": true, "ccp": true, "cnd": true, "con": true,
	"csl": true, "csp": true, "cos": true, "cot": true, "coe": true,
	"cts": true, "ctt": true, "cte": true, "ctr": true, "ctb": true,
	"cpc": true, "cph": true, "crr": true, "crp": true, "cst": true,
	"cou": true, "crt": true, "cov": true, "cre": true, "cur": true,
	"dnc": true, "dtc": true, "dtm": true, "dte": true, "dto": true,
	"dfd": true, "dft": true, "dfe": true, "dgg": true, "dgs": true,
	"dln": true, "dpc": true, "dpt": true, "dsr": true, "drt": true,
	"dis": true, "dbp": true, "dst": true, "dnr": true, "drm": true,
	"dub": true, "edt": true, "edc": true, "edm": true, "elg": true,
	"elt": true, "enj": true, "eng": true, "egr": true, "etr": true,
	"evp": true, "exp": true, "fac": true, "fld": true, "fmd": true,
	"fds": true, "flm": true, "fmp": true, "fmk": true, "fpy": true,
	"frg": true, "fmo": true, "fnd": true, "gis": true, "hnr": true,
	"hst": true, "his": true, "ilu": true, "ill": true, "ins": true,
	"itr": true, "ive": true, "ivr": true, "inv": true, "isb": true,
	"jud": true, "jug": true, "lbr": true, "ldr": true, "lsa": true,
	"led": true, "len": true, "lil": true, "lit": true, "lie": true,
	"lel": true, "let": true, "lee": true, "lbt": true, "lse": true,
	"lso": true, "lgd": true, "ltg": true, "lyr": true, "mfp": true,
	"mfr": true, "mrb": true, "mrk": true, "med": true, "mdc": true,
	"mte": true, "mtk": true, "mod": true, "mon": true, "mcp": true,
	"msd": true, "mus": true, "nrt": true, "osp": true, "opn": true,
	"orm": true, "org": true, "oth": true, "own": true, "pan": true,
	"ppm": true, "pta": true, "pth": true, "pat": true, "prf": true,
	"pma": true, "pht": true, "ptf": true, "ptt": true, "pte": true,
	"plt": true, "pra": true, "pre": true, "prt": true, "pop": true,
	"prm": true, "prc": true, "pro": true, "prn": true, "prs": true,
	"pmn": true, "prd": true, "prp": true, "prg": true, "pdr": true,
	"pfr": true, "prv": true, "pup": true, "pbl": true, "pbd": true,
	"ppt": true, "rdd": true, "rpc": true, "rce": true, "rcd": true,
	"red": true, "ren": true, "rpt": true, "rps": true, "rth": true,
	"rtm": true, "res": true, "rsp": true, "rst": true, "rse": true,
	"rpy": true, "rsg": true, "rsr": true, "rev": true, "rbr": true,
	"sce": true, "sad": true, "aus": true, "scr": true, "scl": true,
	"spy": true, "sec": true, "sll": true, "std": true, "stg": true,
	"sgn": true, "sng": true, "sds": true, "spk": true, "spn": true,
	"sgd": true, "stm": true, "stn": true, "str": true, "stl": true,
	"sht": true, "srv": true, "tch": true, "tcd": true, "tld": true,
	"tlp": true, "ths": true, "trc": true, "trl": true, "tyd": true,
	"tyg": true, "uvp": true, "vdg": true, "vac": true, "wit": true,
	"wde": true, "wdc": true, "wam": true, "wac": true, "wal": true,
	"wat": true, "win": true, "wpr": true, "wst": true}

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

// SetSeries sets the name of the series this book belongs to. A book
// may be in a set or a series, but not both. Note that this is only
// valid for V3 epub books and won't be written out for V2 books.
func (e *EPub) SetSeries(s string) error {
	if e.seriesName != "" {
		return errors.New("series name already set")
	}
	e.seriesName = s
	return nil
}

// SetSet sets the name of the set this book belongs to. A book may be
// in a set or a series, but not both. Note that this is only valid
// for V3 epub books and won't be written out for v2 books.
func (e *EPub) SetSet(s string) error {
	if e.setName != "" {
		return errors.New("set name already set")
	}
	e.setName = s
	return nil
}

// Set the entry number in the set or series of this book. This is
// optional, but if specified it must be a repeating dotted decimal
// number. (like 1.2.3.4.5.6 or 2) This is only valid to set for books
// that have a series or set name attached to them.
func (e *EPub) SetEntryNumber(n string) error {
	n = strings.TrimSpace(n)
	m, err := regexp.MatchString(`^(\d+)(\.\d+)*$`, n)
	if !m || err != nil {
		return errors.New("entry number must match the pattern \\d+(\\.\\d+)*")
	}
	e.entry = n
	return nil
}
