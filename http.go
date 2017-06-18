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
	fmt.Fprintf(w, "{")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key == "cmdline" || kv.Key == "memstats" {
			return
		}
		if !first {
			fmt.Fprintf(w, ",")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "}")
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
<!doctype html>
<html>
	<head>
		<title>DHT search</title>
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1">
		<style>
		* { box-sizing:border-box; }
		body { padding:0;margin:0;color:#666;line-height:1.5;font-size:16px;font-family:sans-serif; }
		ul { list-style:none;padding:0;margin:0; }
		a { color:#000;text-decoration:none; }
		.header { padding:1em;border-bottom:1px solid #555;background-color:#eee; }
		.search { display:flex;float:left;margin:0;padding:0; }
		.search__input { font-size:1.25em;padding:2px; }
		.page { padding:1em; }
		.torrent { margin-bottom:1em; }
		.torrent__name { display:block;overflow:hidden; }
		.search__button, .torrent__magnet { height:32px;width:32px;background: url(data:image/png;base64,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) no-repeat top left; }
		.search__button { display:inline-block;border:none;margin-left:5px;margin-right:5px; }
		.search__button:focus { outline:none; }
		.search__button--active { animation-name:spin;animation-duration:3s;animation-iteration-count:infinite;-webkit-animation-name:spin;-webkit-animation-duration:3s;-webkit-animation-iteration-count:infinite; }
		.torrent__magnet { display:block;float:left;margin-top:4px;margin-right:.5em; }
		.torrent__size, .torrent__file-count, .torrent__seen, .torrent__tags { display:inline-block;padding-right:1em;white-space:nowrap; }
		.tag { display:inline; }
		.files { padding-left:2em;font-family:monospace;font-size:.75em; }
		.files { display:none; }
		.files--active { display:block; }
		.file__size { margin-left:.5em;font-size:.875em; }
		.stats { display:block;margin:0;margin-top:1em; }
		.stats__key, .stats__value { float:left;padding:0;margin:0; }
		.stats__key { clear:left;margin-right:.25em;color:#222;white-space:nowrap; }
		.stats__key:after { content:':'; }
		.stats__value { margin-right:.5em;color:#888; }

		.site-nav:after { content:'';display:table;clear:both; }
		.site-nav__open, .site-nav__close { position:absolute;top:1.25em;right:1em;color:#666; }
		.site-nav__close { left:1em; }
		.site-nav__open:before { content:'☰ ' }
		.site-nav__target { position:fixed;top:0;left:0; }
		.site-nav__target:target + .site-nav__drawer { transform:none; }
		.site-nav__drawer { position:fixed;top:0;right:0;bottom:0;margin:0;padding:1em;padding-top:2em;width:300px;float:right;background-color:#e8e8e8;overflow:visible;z-index:1;transition:0.2s;will-change:tranform;transform:translateX(100%); }
		@keyframes spin { from {transform:rotate(0deg);} to {transform:rotate(360deg);} }
		@-webkit-keyframes spin { from {transform:rotate(0deg);} to {transform:rotate(360deg);} }
		</style>
	</head>
	<body id="body">
		<div class="header">
		<nav class="site-nav">
		<div class="search">
		<input id="search" class="search__input" type="text" name="search" placeholder="Search" />
		<button id="go" class="search__button"></button>
		</div>
		<a href="#trigger:nav" class="site-nav__open">Stats</a>
		<a class="site-nav__target" id="trigger:nav"></a>
		<div class="site-nav__drawer">
        <a href="#0" class="site-nav__close">Close</a>
		<dl id="stats" class="stats"></dl>
		</div>
		</nav>
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
var processResponse = function (resp) {
	if (resp.status === 200 || resp.status === 0) {
		return resp.json()
	} else {
		return new Error(resp.statusText)
	}
}
var search = function (term) {
	bEl.classList.add('search__button--active')
	var query = ''
	var tIdx = term.indexOf('tag:')
	if (tIdx >= 0) {
		query = 'tag=' + term.slice(tIdx+4).trim()
	} else {
		query = 'q=' + term.trim()
    }
	fetch('/search?' + query)
	.then(processResponse)
	.then(function (data) {
		var pre = [
		'<p>Displaying ', data.length, ' torrents.</p>',
		'<ul id="results">'
		].join('')
		var torrents = data.map(function (t) {
			var magnet = 'magnet:?xt=urn:btih:' + t.infohash
			return [
			'<li class="torrent">',
			'<a class="torrent__magnet" href="', magnet, '"></a>',
			'<a class="torrent__name" href="', magnet, '">', t.name, '</a>',
			'<span class="torrent__size">Size: ', humanSize(t.size), '</span>',
			'<span class="torrent__seen">Last seen: <time datetime="', t.seen, '">', new Date(t.seen).toLocaleString(), '</time></span>',
			'<span class="torrent__tags">Tags: ',
			t.tags.map(function (g) { return '<a class="tag" href="/tags/' + g + '">' + g + '</a>' }).join(', '),
			'</span>',
			t.files.length === 0 ? '' : ['<a class="torrent__file-count toggler" href="#">Files: ', t.files.length, '</a>'].join(''),
			'<ul class="files">',
			t.files.map(function (f) {
				return [
				'<li class="files__file file">',
				'<span class="file__path">', f.path, '</span>',
				'<span class="file__size">[', humanSize(f.size), ']</span>',
				'</li>'
				].join('')
			}).join(''),
			'</ul>',
			'</li>'
			].join('')
		}).join('')
		var post = '</ul>'
		pEl.innerHTML = pre + torrents + post
		bEl.classList.remove('search__button--active')
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
	if (term.length > 2) {
		console.log('Search term: ', term)
		search(term)
	}
})
sEl.addEventListener('keyup', function (e) {
	if (e.keyCode === 13) {
		bEl.click()
	}
})
var statsEl = document.getElementById('stats')
var getStats = function () {
	fetch('/stats')
	.then(processResponse)
	.then(function (data) {
		statsEl.innerHTML = Object.keys(data).map(function (k) {
			return [
			'<dt class="stats__key">',
			k.replace(/_/g,' '),
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
