package store

// Look is the parametric ghost-pet appearance. The hash matches web/js/pets.js
// so a name always draws the same pet on the server and in the browser.
type Look struct {
	Ear     string `json:"ear"`
	Color   string `json:"color"`
	Tool    string `json:"tool"`
	Glasses bool   `json:"glasses"`
}

var (
	ears   = []string{"cat", "bunny", "bear", "mouse", "fox", "sprout"}
	colors = []string{"mint", "blue", "yellow", "pink", "lav", "peach"}
	tools  = []string{"cube", "paper", "wrench", "pencil", "mug", "lens", "none"}
)

func known(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// HashName is the pets.js hash: FNV-1a over UTF-16 code units, then the
// xorshift finalizer. BMP characters (which covers Chinese names) match runes.
func HashName(s string) uint32 {
	h := uint32(2166136261)
	for _, r := range s {
		h ^= uint32(r)
		h *= 16777619
	}
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return h
}

// GenLook builds a pet from a name. salt is the "换一个" counter; 0 is the
// canonical look for that name.
func GenLook(name string, salt int) Look {
	s := name
	if salt > 0 {
		s = name + "#" + itoa(salt)
	}
	h := HashName(s)
	return Look{
		Ear:     ears[h%6],
		Color:   colors[(h/6)%6],
		Tool:    tools[(h/36)%7],
		Glasses: (h/252)%10 < 3,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// NormalizeLook checks a client-supplied look and fills defaults.
func NormalizeLook(l Look) (Look, error) {
	if l.Ear == "" {
		l.Ear = "cat"
	}
	if l.Color == "" {
		l.Color = "lav"
	}
	if l.Tool == "" {
		l.Tool = "none"
	}
	if !known(ears, l.Ear) || !known(colors, l.Color) || !known(tools, l.Tool) {
		return Look{}, invalid("形象参数不对")
	}
	return l, nil
}

func ValidColor(c string) bool { return known(colors, c) }

func Colors() []string { return append([]string(nil), colors...) }
