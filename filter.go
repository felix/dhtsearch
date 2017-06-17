package main

import (
	"regexp"
)

var tags = map[string]string{
	"flac":        `\.flac$`,
	"episode":     "(season|episode|s[0-9]{2}e[0-9]{2})",
	"1080":        "1080",
	"720":         "720",
	"hd":          "hd|720|1080",
	"bdrip":       "bdrip",
	"adult":       `(xxx|f.ck|p(orn|ussy)|censor|sex|urbat|a(ss|nal)\s|(di|co)ck|esbian|milf|lust|gay)|erotic|18(\+|yr)`,
	"dvdrip":      "dvdrip",
	"ebook":       "epub",
	"application": `\.(apk|exe|msi|dmg)$`,
	"android":     `\.apk$`,
	"apple":       `\.dmg$`,
	"subtitles":   `\.s(rt|ub)$`,
	"archive":     `\.(zip|rar|p7|tgz|bz2)$`,
	"video":       `\.(3g2|3gp|amv|asf|avi|drc|f4a|f4b|f4p|f4v|flv|gif|gifv|m2v|m4p|m4v|mkv|mng|mov|mp2|mp4|mpe|mpeg|mpg|mpv|mxf|net|nsv|ogv|qt|rm|rmvb|roq|svi|vob|webm|wmv|yuv)$`,
	"audio":       `\.(aa|aac|aax|act|aiff|amr|ape|au|awb|dct|dss|dvf|flac|gsm|iklax|ivs|m4a|m4b|mmf|mp3|mpc|msv|ogg|opus|ra|raw|sln|tta|vox|wav|wma|wv)$`,
	"document":    `\.(cbr|cbz|cb7|cbt|cba|epub|djvu|fb2|ibook|azw.|lit|prc|mobi|pdb|pdb|oxps|xps)$`,
	"font":        `(font|\.(ttf|fon)$)`,
}

var tagREs map[string]*regexp.Regexp

// Filter on words, existing
func initTagRegexps() {
	tagREs = make(map[string]*regexp.Regexp)
	for tag, re := range tags {
		tagREs[tag] = regexp.MustCompile("(?i)" + re)
	}
}

func tagTorrent(t *Torrent) {
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
		t.Tags = append(t.Tags, tt)
	}
}
