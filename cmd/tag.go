package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"src.userspace.com.au/dhtsearch/models"
)

// Default tags, can be supplimented or overwritten by config
var tags = map[string]string{
	"flac":        `\.flac$`,
	"episode":     "(season|episode|s[0-9]{2}e[0-9]{2})",
	"1080":        "1080",
	"720":         "720",
	"hd":          "hd|720|1080|4k",
	"rip":         "(bdrip|dvd[- ]?rip|dvdmux|br[- ]?rip|dvd[-]?r|web[- ]?dl|hdrip)",
	"xxx":         `(xxx|p(orn|ussy)|censor|sex|urbat|a(ss|nal)|o(rgy|gasm)|(fu|di|co)ck|esbian|milf|lust|gay)|rotic|18(\+|yr)|hore|hemale|virgin`,
	"ebook":       "epub",
	"application": `\.(apk|exe|msi|dmg)$`,
	"android":     `\.apk$`,
	"apple":       `\.dmg$`,
	"subtitles":   `\.s(rt|ub)$`,
	"archive":     `\.(zip|rar|p7|tgz|bz2|iso)$`,
	"video":       `(\.(3g2|3gp|amv|asf|avi|drc|f4a|f4b|f4p|f4v|flv|gif|gifv|m2v|m4p|m4v|mkv|mng|mov|mp2|mp4|mpe|mpeg|mpg|mpv|mxf|net|nsv|ogv|qt|rm|rmvb|roq|svi|vob|webm|wmv|yuv)$|divx|x264|x265)`,
	"audio":       `\.(aa|aac|aax|act|aiff|amr|ape|au|awb|dct|dss|dvf|flac|gsm|iklax|ivs|m4a|m4b|mmf|mp3|mpc|msv|ogg|opus|ra|raw|sln|tta|vox|wav|wma|wv)$`,
	"document":    `\.(cbr|cbz|cb7|cbt|cba|epub|djvu|fb2|ibook|azw.|lit|prc|mobi|pdb|pdb|oxps|xps|pdf)$`,
	"bootleg":     `(camrip|hdts|[-. ](ts|tc)[-. ]|hdtc)`,
	"screener":    `(bd[-]?scr|screener|dvd[-]?scr|r5)`,
	"font":        `(font|\.(ttf|fon|otf)$)`,
}

func mergeCharacterTagREs(tagREs map[string]*regexp.Regexp) error {
	var err error
	// Add character classes
	for cc := range unicode.Scripts {
		if cc == "Latin" || cc == "Common" {
			continue
		}
		className := strings.ToLower(cc)
		// Test for 3 or more characters per character class
		tagREs[className], err = regexp.Compile(fmt.Sprintf(`(?i)\p{%s}{3,}`, cc))
		if err != nil {
			return err
		}
	}
	return nil
}

func mergeTagRegexps(tagREs map[string]*regexp.Regexp, tags map[string]string) error {
	var err error
	for tag, re := range tags {
		tagREs[tag], err = regexp.Compile("(?i)" + re)
		if err != nil {
			return err
		}
	}
	return nil
}

func tagTorrent(t models.Torrent, tagREs map[string]*regexp.Regexp) (tags []string) {
	ttags := make(map[string]bool)

	for tag, re := range tagREs {
		if re.MatchString(t.Name) {
			ttags[tag] = true
		}
		for _, f := range t.Files {
			if re.MatchString(f.Path) {
				ttags[tag] = true
			}
		}
	}
	// Make unique
	for tt := range ttags {
		tags = append(tags, tt)
	}
	return tags
}

func hasTag(t models.Torrent, tag string) bool {
	for _, t := range t.Tags {
		if tag == t {
			return true
		}
	}
	return false
}
