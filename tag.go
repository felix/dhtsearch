package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Default tags, can be supplimented or overwritten by config
var tags = map[string]string{
	"flac":        `\.flac$`,
	"episode":     "(season|episode|s[0-9]{2}e[0-9]{2})",
	"1080":        "1080",
	"720":         "720",
	"hd":          "hd|720|1080",
	"bdrip":       "bdrip",
	"adult":       `(xxx|p(orn|ussy)|censor|sex|urbat|a(ss|nal)|o(rgy|gasm)|(fu|di|co)ck|esbian|milf|lust|gay)|rotic|18(\+|yr)`,
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
	// Add character classes
	for cc, _ := range unicode.Scripts {
		if cc == "Latin" || cc == "Common" {
			continue
		}
		className := strings.ToLower(cc)
		// Test for 3 or more characters per character class
		tagREs[className] = regexp.MustCompile(fmt.Sprintf(`(?i)\p{%s}{3,}`, cc))
	}
	// Merge user tags
	for tag, re := range Config.Tags {
		if !Config.Quiet {
			fmt.Printf("Adding user tag: %s = %s\n", tag, re)
		}
		tagREs[tag] = regexp.MustCompile("(?i)" + re)
	}
}

func createTag(tag string) (tagId int, err error) {
	err = DB.QueryRow(sqlSelectTag, tag).Scan(&tagId)
	if err == nil {
		if Config.Debug {
			fmt.Printf("Found existing tag %s\n", tag)
		}
	} else {
		err = DB.QueryRow(sqlInsertTag, tag).Scan(&tagId)
		if err != nil {
			fmt.Println(err)
			return -1, err
		}
		if Config.Debug {
			fmt.Printf("Created new tag %s\n", tag)
		}
	}
	return tagId, nil
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

func hasTag(t Torrent, tag string) bool {
	for _, t := range t.Tags {
		if tag == t {
			return true
		}
	}
	return false
}

const (
	sqlSelectTag = `select id from tags where name = $1`
	sqlInsertTag = `insert into tags (name) values ($1) returning id`
)
