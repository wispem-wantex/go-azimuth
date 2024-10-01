package phonemes

import (
	"fmt"
	"strings"
)

var PREFIXES = [256]string{
	"doz", "mar", "bin", "wan", "sam", "lit", "sig", "hid",
	"fid", "lis", "sog", "dir", "wac", "sab", "wis", "sib",
	"rig", "sol", "dop", "mod", "fog", "lid", "hop", "dar",
	"dor", "lor", "hod", "fol", "rin", "tog", "sil", "mir",
	"hol", "pas", "lac", "rov", "liv", "dal", "sat", "lib",
	"tab", "han", "tic", "pid", "tor", "bol", "fos", "dot",
	"los", "dil", "for", "pil", "ram", "tir", "win", "tad",
	"bic", "dif", "roc", "wid", "bis", "das", "mid", "lop",
	"ril", "nar", "dap", "mol", "san", "loc", "nov", "sit",
	"nid", "tip", "sic", "rop", "wit", "nat", "pan", "min",
	"rit", "pod", "mot", "tam", "tol", "sav", "pos", "nap",
	"nop", "som", "fin", "fon", "ban", "mor", "wor", "sip",
	"ron", "nor", "bot", "wic", "soc", "wat", "dol", "mag",
	"pic", "dav", "bid", "bal", "tim", "tas", "mal", "lig",
	"siv", "tag", "pad", "sal", "div", "dac", "tan", "sid",
	"fab", "tar", "mon", "ran", "nis", "wol", "mis", "pal",
	"las", "dis", "map", "rab", "tob", "rol", "lat", "lon",
	"nod", "nav", "fig", "nom", "nib", "pag", "sop", "ral",
	"bil", "had", "doc", "rid", "moc", "pac", "rav", "rip",
	"fal", "tod", "til", "tin", "hap", "mic", "fan", "pat",
	"tac", "lab", "mog", "sim", "son", "pin", "lom", "ric",
	"tap", "fir", "has", "bos", "bat", "poc", "hac", "tid",
	"hav", "sap", "lin", "dib", "hos", "dab", "bit", "bar",
	"rac", "par", "lod", "dos", "bor", "toc", "hil", "mac",
	"tom", "dig", "fil", "fas", "mit", "hob", "har", "mig",
	"hin", "rad", "mas", "hal", "rag", "lag", "fad", "top",
	"mop", "hab", "nil", "nos", "mil", "fop", "fam", "dat",
	"nol", "din", "hat", "nac", "ris", "fot", "rib", "hoc",
	"nim", "lar", "fit", "wal", "rap", "sar", "nal", "mos",
	"lan", "don", "dan", "lad", "dov", "riv", "bac", "pol",
	"lap", "tal", "pit", "nam", "bon", "ros", "ton", "fod",
	"pon", "sov", "noc", "sor", "lav", "mat", "mip", "fip",
}
var PREFIX_LOOKUP = map[string]uint64{}

var SUFFIXES = [256]string{
	"zod", "nec", "bud", "wes", "sev", "per", "sut", "let",
	"ful", "pen", "syt", "dur", "wep", "ser", "wyl", "sun",
	"ryp", "syx", "dyr", "nup", "heb", "peg", "lup", "dep",
	"dys", "put", "lug", "hec", "ryt", "tyv", "syd", "nex",
	"lun", "mep", "lut", "sep", "pes", "del", "sul", "ped",
	"tem", "led", "tul", "met", "wen", "byn", "hex", "feb",
	"pyl", "dul", "het", "mev", "rut", "tyl", "wyd", "tep",
	"bes", "dex", "sef", "wyc", "bur", "der", "nep", "pur",
	"rys", "reb", "den", "nut", "sub", "pet", "rul", "syn",
	"reg", "tyd", "sup", "sem", "wyn", "rec", "meg", "net",
	"sec", "mul", "nym", "tev", "web", "sum", "mut", "nyx",
	"rex", "teb", "fus", "hep", "ben", "mus", "wyx", "sym",
	"sel", "ruc", "dec", "wex", "syr", "wet", "dyl", "myn",
	"mes", "det", "bet", "bel", "tux", "tug", "myr", "pel",
	"syp", "ter", "meb", "set", "dut", "deg", "tex", "sur",
	"fel", "tud", "nux", "rux", "ren", "wyt", "nub", "med",
	"lyt", "dus", "neb", "rum", "tyn", "seg", "lyx", "pun",
	"res", "red", "fun", "rev", "ref", "mec", "ted", "rus",
	"bex", "leb", "dux", "ryn", "num", "pyx", "ryg", "ryx",
	"fep", "tyr", "tus", "tyc", "leg", "nem", "fer", "mer",
	"ten", "lus", "nus", "syl", "tec", "mex", "pub", "rym",
	"tuc", "fyl", "lep", "deb", "ber", "mug", "hut", "tun",
	"byl", "sud", "pem", "dev", "lur", "def", "bus", "bep",
	"run", "mel", "pex", "dyt", "byt", "typ", "lev", "myl",
	"wed", "duc", "fur", "fex", "nul", "luc", "len", "ner",
	"lex", "rup", "ned", "lec", "ryd", "lyd", "fen", "wel",
	"nyd", "hus", "rel", "rud", "nes", "hes", "fet", "des",
	"ret", "dun", "ler", "nyr", "seb", "hul", "ryl", "lud",
	"rem", "lys", "fyn", "wer", "ryc", "sug", "nys", "nyl",
	"lyn", "dyn", "dem", "lux", "fed", "sed", "bec", "mun",
	"lyr", "tes", "mud", "nyt", "byr", "sen", "weg", "fyr",
	"mur", "tel", "rep", "teg", "pec", "nel", "nev", "fes",
}
var SUFFIX_LOOKUP = map[string]uint64{}

func init() {
	for i, v := range PREFIXES {
		PREFIX_LOOKUP[v] = uint64(i)
	}
	for i, v := range SUFFIXES {
		SUFFIX_LOOKUP[v] = uint64(i)
	}
}

func IntToPhonemeNonGalaxy(i uint64) string {
	if i < 0x1_00_00 {
		pfix := PREFIXES[i/0x1_00]
		sfix := SUFFIXES[i%0x1_00]
		return pfix + sfix
	}
	return fmt.Sprintf("%s-%s", IntToPhonemeNonGalaxy(i/0x1_00_00), IntToPhonemeNonGalaxy(i%0x1_00_00))
}

func IntToPhonemeQ(i uint64) string {
	if i < 0x1_00 {
		return SUFFIXES[i]
	}

	return IntToPhonemeNonGalaxy(i)
}

// Returns the value and a bool indicating whether the conversion succeeded (whether it was successful)
func PhonemeQToInt(s string) (uint64, bool) {
	if len(s) == 3 {
		val, is_ok := SUFFIX_LOOKUP[s]
		return val, is_ok
	}

	ret := uint64(0)
	for _, pfix_sfix := range strings.Split(s, "-") {
		if len(pfix_sfix) != 6 {
			return 0, false
		}
		ret <<= 16
		pfix, is_ok := PREFIX_LOOKUP[pfix_sfix[:3]]
		if !is_ok {
			return 0, false
		}
		ret += pfix << 8
		sfix, is_ok := SUFFIX_LOOKUP[pfix_sfix[3:]]
		if !is_ok {
			return 0, false
		}
		ret += sfix
	}
	return ret, true
}

func IntToPhoneme(i uint64) string {
	if i < 0x1_0000_0000 {
		return IntToPhonemeQ(uint64(Scramble(uint32(i))))
	}
	j := (i & 0xffff_ffff_0000_0000) + uint64(Scramble(uint32(i&0xffff_ffff)))
	return IntToPhonemeQ(j)
}

func PhonemeToInt(s string) (uint64, bool) {
	i, is_ok := PhonemeQToInt(s)
	if !is_ok {
		return 0, false
	}
	if i < 0x1_0000_0000 {
		return uint64(Unscramble(uint32(i))), true
	}
	return (i & 0xffff_ffff_0000_0000) + uint64(Unscramble(uint32(i&0xffff_ffff))), true
}
