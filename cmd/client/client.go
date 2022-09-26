package main

import (
	"context"
	"fmt"

	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	util "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	ctx := context.Background()
	mgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(rcmgr.InfiniteLimits))
	if err != nil {
		panic(err)
	}
	var dht *kaddht.IpfsDHT
	h, err := libp2p.New(
		libp2p.UserAgent("antares-client/0.1.0"),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			dht, err = kaddht.New(ctx, h)
			return dht, err
		}),
		libp2p.ResourceManager(mgr),
	)
	if err != nil {
		panic(err)
	}

	network := bsnet.NewFromIpfsHost(h, dht)
	ds := dssync.MutexWrap(datastore.NewMapDatastore())
	bstore := blockstore.NewBlockstore(ds)
	bs := bitswap.New(ctx, network, bstore)

	c, err := cid.Decode("QmUKvQPrKrWERzKaRSbGvgjJFL9Pne6j9YbYtLfrL9i4hF")

	peerID, err := peer.Decode("12D3KooWK8ioLrie7u9dw8DdLBL7HYQCKAjQV4zaWVWXgpE2XRni")
	if err != nil {
		panic(err)
	}

	ma, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/2002")
	if err != nil {
		panic(err)
	}

	pi := peer.AddrInfo{
		ID:    peerID,
		Addrs: []multiaddr.Multiaddr{ma},
	}

	if err = h.Connect(ctx, pi); err != nil {
		panic(err)
	}

	blk, err := bs.GetBlock(ctx, c)
	if err != nil {
		panic(err)
	}

	fmt.Println("Got block", blk.Cid(), string(blk.RawData()))

	x := cid.NewCidV0(util.Hash(blk.RawData()))
	if err != nil {
		panic(err)
	}
	fmt.Println("Got block", x)
}
