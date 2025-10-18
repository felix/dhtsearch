package dht

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

const (
	ip4AddrLength          = 4
	ip6AddrLength          = 16
	portLength             = 2
	compactIP4Length       = ip4AddrLength + portLength
	compactIP6Length       = ip6AddrLength + portLength
	CompactNodeInfoLength  = infohash.Length + compactIP4Length // 26 bytes
	CompactNode6InfoLength = infohash.Length + compactIP6Length // 38 bytes
)

type NodeAddr netip.AddrPort

// Like AddrPort UnmarshalBinary except port is big-endian
func (n *NodeAddr) UnmarshalBinary(b []byte) error {
	var addr netip.Addr
	var offset int
	var ok bool
	if len(b) > compactIP4Length {
		addr, ok = netip.AddrFromSlice(b[:ip6AddrLength])
		offset = ip6AddrLength
	} else {
		addr, ok = netip.AddrFromSlice(b[:ip4AddrLength])
		offset = ip4AddrLength
	}
	if !ok {
		return errors.New("failed to parse node address")
	}
	ap := netip.AddrPortFrom(addr, binary.BigEndian.Uint16(b[offset:]))
	*n = NodeAddr(ap)
	return nil
}

// Like AddrPort MarshalBinary except port is big-endian
func (n NodeAddr) MarshalBinary() ([]byte, error) {
	ap := netip.AddrPort(n)
	b, err := ap.Addr().MarshalBinary()
	if err != nil {
		return nil, err
	}
	return binary.Append(b, binary.BigEndian, ap.Port())
}

type CompactNodeList struct {
	Nodes []*Node
}

func (c *CompactNodeList) UnmarshalBencode(b []byte) error {
	var in []byte
	if err := bencode.Unmarshal(b, &in); err != nil {
		return err
	}
	if mod := len(in) % CompactNodeInfoLength; mod != 0 {
		return fmt.Errorf("CompactNodeList trailing %d bytes", mod)
	}
	//fmt.Println("chunking", len(in), "into chunks of", CompactNodeInfoLength)
	for chunk := range slices.Chunk(in, CompactNodeInfoLength) {
		if len(chunk) == CompactNodeInfoLength {
			n := new(Node)
			if err := n.UnmarshalBinary(chunk); err != nil {
				continue
			}
			c.Nodes = append(c.Nodes, n)
		}
	}
	return nil
}

func (c *CompactNodeList) MarshalBencode() ([]byte, error) {
	var bb []byte
	for _, n := range c.Nodes {
		b, err := n.MarshalBinary()
		if err != nil {
			return nil, err
		}
		bb = append(bb, b...)
	}
	out := fmt.Appendf([]byte{}, "%d:", len(bb))
	return append(out, bb...), nil
}

type CompactNode6List struct {
	Nodes []*Node
}

func (c *CompactNode6List) UnmarshalBencode(b []byte) error {
	var in []byte
	if err := bencode.Unmarshal(b, &in); err != nil {
		return err
	}
	//fmt.Println("nodes6", len(in), len(b), "in", hex.EncodeToString(b))
	// if in == "" {
	// 	return nil
	// }
	if mod := len(in) % CompactNode6InfoLength; mod != 0 {
		return fmt.Errorf("CompactNode6List trailing %d bytes", mod)
	}
	for chunk := range slices.Chunk([]byte(in), CompactNode6InfoLength) {
		if len(chunk) == CompactNode6InfoLength {
			n := new(Node)
			if err := n.UnmarshalBinary(chunk); err != nil {
				return err
			}
			c.Nodes = append(c.Nodes, n)
		}
	}
	return nil
}

func (c *CompactNode6List) MarshalBencode() ([]byte, error) {
	var bb []byte
	for _, n := range c.Nodes {
		b, err := n.MarshalBinary()
		if err != nil {
			return nil, err
		}
		bb = append(bb, b...)
	}
	out := fmt.Appendf([]byte{}, "%d:", len(bb))
	return append(out, bb...), nil
}
