package main

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(200)
	w.Write(html)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public")
	w.WriteHeader(200)
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key == "cmdline" || kv.Key == "memstats" {
			return
		}
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if q := r.URL.Query().Get("q"); q != "" {
		torrents, err := torrentsByName(q)
		if err != nil {
			w.WriteHeader(500)
			fmt.Printf("Error: %q\n", err)
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(torrents)
		return
	}

	if tag := r.URL.Query().Get("tag"); tag != "" {
		torrents, err := torrentsByTag(tag)
		if err != nil {
			w.WriteHeader(500)
			fmt.Printf("Error: %q\n", err)
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(torrents)
		return
	}

	w.WriteHeader(406)
	json.NewEncoder(w).Encode("Query required")
}

var html = []byte(`
<html>
	<head>
		<title>DHT search</title>
		<style>
		body { padding:0;margin:0;color:#666;line-height:1.5;font-size:16px; }
		.header { padding:1em;border-bottom:1px solid #555; }
		input { padding:2px; }
		.page { padding:1em 2em; }
		ul { list-style:none;padding:0;margin:0; }
		.torrent { margin-bottom:1em; }
		.torrent__name { display:block; }
		.torrent__magnet { display:block; }
		.torrent__size, .torrent__file-count, .torrent__seen, .torrent__tags { padding-right:1em; }
		.tag { display:inline;text-decoration:none; }
		.files { padding-left:2em;font-family:monospace;font-size:.75em; }
		.files { display:none; }
		.files--active { display:block; }
		.file__size { margin-left:.5em;font-size:.875em; }
		.stats { display:block;margin:0;margin-top:1em; }
		.stats__key, .stats__value { display:inline-block;font-size:.75em;padding:0;margin:0; }
		.stats__key { margin-right:.25em;color:#222; }
		.stats__key:after { content:':'; }
		.stats__value { margin-right:.5em;color:#888; }
		</style>
	</head>
	<body id="body">
		<div class="header">
		<input id="search" type="text" name="search" placeholder="Search" />
		<button id="go">Go</button>
		<dl id="stats" class="stats"></dl>
		</div>
		<div id="page" class="page">
		</div>
		<script>
var humanSize = function (bytes) {
	if (bytes === 0) {
		return '0 Bytes'
	}
	var sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
	var sizeIndex = parseInt(Math.floor(Math.log(bytes) / Math.log(1024)), 10)
	if (sizeIndex >= sizes.length) {
		sizeIndex = sizes.length - 1
	}
	return (bytes / Math.pow(1024, sizeIndex)).toFixed(0) + sizes[sizeIndex]
}
var search = function (term) {
	var oldButton = bEl.innerHTML
	bEl.innerHTML = 'Searching'
	var query = ''
	var tIdx = term.indexOf('tag:')
	if (tIdx >= 0) {
		query = 'tag=' + term.slice(tIdx+4).trim()
	} else {
		query = 'q=' + term.trim()
    }
	fetch('/search?' + query)
	.then(function (resp) {
		if (resp.status === 200 || resp.status === 0) {
			return resp.json()
		} else {
			return new Error(resp.statusText)
		}
	})
	.then(function (data) {
		var pre = [
		'<p>Displaying ', data.length, ' torrents.</p>',
		'<ul id="results">'
		].join('')
		var torrents = data.map(function (t) {
			var magnet = 'magnet:?xt=urn:btih:' + t.infohash
			var pre = [
			'<li class="torrent">',
			'<a class="torrent__name" href="', magnet, '">', t.name, '</a>',
			'<span class="torrent__magnet">', magnet, '</span>',
			'<span class="torrent__size">Size:&nbsp;', humanSize(t.size), '</span>',
			'<span class="torrent__seen">Last&nbsp;seen:&nbsp;<time datetime="', t.seen, '">', new Date(t.seen).toLocaleString(), '</time></span>'
			].join('')
			var tags = ''
			if (t.tags.length) {
				tags = [
				'<span class="torrent__tags">Tags: ',
				t.tags.map(function (g) { return '<a class="tag" href="/tags/' + g + '">' + g + '</a>' }).join(',&nbsp;'),
				'</span>'
				].join('')
			}
			var files = ''
			if (t.files.length) {
				var fHtml = t.files.map(function (f) {
					return [
					'<li class="files__file file">',
					'<span class="file__path">', f.path, '</span>',
					'<span class="file__size">[', humanSize(f.size), ']</span>',
					'</li>'
					].join('')
				})
				files = [
				'<span class="torrent__file-count">Files:&nbsp;', t.files.length, '</span>',
				'<a class="toggler" href="#">toggle files</a><ul class="files">',
				fHtml.join(''),
				'</ul>'].join('')
			}
			var post = [
			'</li>'
			].join('')
			return pre + tags + files + post
		}).join('')
		var post = '</ul>'
		bEl.innerHTML = oldButton
		pEl.innerHTML = pre + torrents + post
		var togglers = document.getElementsByClassName('toggler')
		for (var i = 0; i < togglers.length; i += 1) {
			var el = togglers[i]
			el.addEventListener('click', function (e) {
				e.preventDefault()
				e.target.nextElementSibling.classList.toggle('files--active')
			})
		}
	})
}
var pEl = document.getElementById('page')
var sEl = document.getElementById('search')
var bEl = document.getElementById('go')
bEl.addEventListener('click', function () {
	var term = sEl.value
	console.log('Search term: ', term)
	search(term)
})
sEl.addEventListener('keyup', function (e) {
	if (e.keyCode === 13) {
		bEl.click()
	}
})
var statsEl = document.getElementById('stats')
var getStats = function () {
	fetch('/stats')
	.then(function (resp) {
		if (resp.status === 200 || resp.status === 0) {
			return resp.json()
		} else {
			return new Error(resp.statusText)
		}
	})
	.then(function (data) {
		statsEl.innerHTML = Object.keys(data).map(function (k) {
			return [
			'<dt class="stats__key">',
			k.replace(/_/g,'&nbsp;'),
			'</dt><dd class="stats__value">',
			k.indexOf('bytes') === -1 ? data[k] : humanSize(data[k]),
			'</dd>'
			].join('')
		}).join('')
		setTimeout(getStats, 5000)
	})
}
getStats()
		</script>
	</body>
</html>
`)
