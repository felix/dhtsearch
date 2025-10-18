package dht

import (
	"bytes"
	"net/netip"
	"testing"

	"github.com/neilotoole/slogt"

	"userspace.com.au/dhtsearch/infohash"
)

func TestTable(t *testing.T) {
	id := bytes.Repeat([]byte{0}, 20)
	tbl := NewTable(infohash.ID(id))
	tbl.log = slogt.New(t)

	var nodes []*Node

	for range 100 {
		n := &Node{
			ID: infohash.NewRandomID(),
		}
		if tbl.Add(n) {
			nodes = append(nodes, n)
		}
	}

	t.Run("updating", func(t *testing.T) {
		t.Parallel()
		for _, n := range nodes {
			tbl.SetSeen(n.ID)
		}
	})
	t.Run("lookup", func(t *testing.T) {
		t.Parallel()
		for _, n := range nodes {
			tbl.Has(n.ID)
		}
	})
	t.Run("remove add", func(t *testing.T) {
		t.Parallel()
		for _, n := range nodes {
			tbl.Remove(n.ID)
			tbl.Add(n)
		}
	})
}

func TestTableExport(t *testing.T) {
	nodes := []Node{
		{
			ID:       infohash.MustParseString("0000000000000000000000000000000000000000"),
			AddrPort: netip.MustParseAddrPort("[2001:19f0:5:6d01:5400:2ff:feec:644a]:6881")},
		{
			ID:       infohash.MustParseString("c7fb3eab2d33331610075ed0322f6cebc5351aed"),
			AddrPort: netip.MustParseAddrPort("[2607:9000:3000:33:d5fe:c61c:6adc:6b79]:17980")},
		{
			ID:       infohash.MustParseString("c7fba7fc5ec5dfd49cff98b31b68e37b0a08fd84"),
			AddrPort: netip.MustParseAddrPort("[240e:3b3:9613:52f0:215:5dff:fe0a:8971]:61128")},
		{
			ID:       infohash.MustParseString("c7fc64a9ebb14481981529ba200a40d00643eb2a"),
			AddrPort: netip.MustParseAddrPort("[2001:da8:e000:3002:20c:29ff:fea3:7226]:10301")},
		{
			ID:       infohash.MustParseString("c5198df3d46a32799ac34436a4737d7b3bb2ff6e"),
			AddrPort: netip.MustParseAddrPort("[240e:b8f:966e:9e00::bf9]:63219")},
		{
			ID:       infohash.MustParseString("c1ea1df5ce162836c2080bdc94a127a90cfad24a"),
			AddrPort: netip.MustParseAddrPort("[2a01:e0a:d5b:c690:211:32ff:fe26:7445]:51413")},
		{
			ID:       infohash.MustParseString("c17f24479d8c5ad3ad7a731a600ff323ecb26ebd"),
			AddrPort: netip.MustParseAddrPort("[2001:e68:542f:5959:11b7:2243:527d:1c86]:19264")},
	}
	id := infohash.MustParseString("c4bee4bf16dd88527f63cab902b1111111111111")
	tbl := NewTable(id)
	tbl.log = slogt.New(t)
	for _, n := range nodes {
		tbl.Add(&n)
	}
	if tbl.Count() != len(nodes) {
		t.Fatalf("got %d, want %d", tbl.Count(), len(nodes))
	}

	buf := new(bytes.Buffer)
	n1, err := tbl.WriteTo(buf)
	if err != nil {
		t.Fatal(err)
	}
	tbl2 := &Table{
		root: newBucket(),
		k:    8,
	}
	tbl2.log = slogt.New(t)
	n2, err := tbl2.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Errorf("got %d, want %d", n2, n1)
	}
	for _, n := range nodes {
		if !tbl.Has(n.ID) {
			t.Errorf("missing id %s", n.ID)
		}
	}
}
