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
	w.Header().Set("Cache-Control", "no-cache")
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
		body { padding:0;margin:0; }
		.header { padding:1em;border-bottom:1px solid #555; }
		input { padding:2px; }
		.page { padding:2em; }
		ul { list-style:none;padding:0;margin:0; }
		.torrent { margin-bottom:1em; }
		.torrent__name { display:block; }
		.torrent__size, .torrent__file-count, .torrent__seen { padding-right:1em; }
		.torrent__tags { display:block; }
		.tag { display:inline-block;margin-right:2px;padding:3px 5px;border:1px solid #ddd;border-radius:3px; }
		.files { padding-left:2em; }
		.files__file { display:none; }
		</style>
	</head>
	<body id="body">
		<div class="header">
		<input id="search" type="text" name="search" />
		<button id="go">Search</button>
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
	return (bytes / Math.pow(1024, sizeIndex)).toFixed(0) + ' ' + sizes[sizeIndex]
}
var months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
var humanDate = function (iso) {
	var d = new Date(iso)
	return [d.getDate(), months[d.getMonth()], d.getFullYear()].join(' ')
}
var search = function (term) {
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
			var pre = [
			'<li class="torrent">',
			'<a class="torrent__name" href="magnet:?xt=urn:btih:', t.infohash, '">', t.name, '</a>',
			'<span class="torrent__size">', humanSize(t.size), '</span>',
			'<span class="torrent__file-count">', t.files.length, ' files</span>',
			'<span class="torrent__seen">Last seen: <time datetime="', t.seen, '">', humanDate(t.seen), '</time></span>',
			].join('')
			var tags = ''
			if (t.tags.length) {
				tags = [
				'<span class="torrent__tags">',
				t.tags.map(function (g) { return '<a class="tag" href="/tags/' + g + '">' + g + '</a>' }).join(''),
				'</span>'
				].join('')
			}
			var files = ''
			if (t.files.length) {
				var fHtml = t.files.map(function (f) {
					return [
					'<li class="files__file file">',
					'<span class="file__path">', f.path, '</span>',
					'<span class="file__size">', humanSize(f.size), '</span>',
					'</li>'
					].join('')
				})
				files = '<ul class="files">' + fHtml.join('') + '</ul>'
			}
			var post = [
			'</li>'
			].join('')
			return pre + tags + files + post
		}).join('')
		var post = '</ul>'
		pEl.innerHTML = pre + torrents + post
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
		</script>
	</body>
</html>
`)
