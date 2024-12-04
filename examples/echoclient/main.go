package main

import (
	"context"
	"encoding/hex"
	"flag"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/threefoldtech/libp2p-relay/client"

	logging "github.com/ipfs/go-log/v2"
)

const Protocol = "/echo/1.0.0"

func main() {

	var hexPSK string
	var relay string
	var remotePeerID string
	var verbose bool
	flag.StringVar(&hexPSK, "psk", "", "32 bytes network PSK in hex")
	flag.StringVar(&relay, "relay", "", "relay multi-address")
	flag.StringVar(&remotePeerID, "remote", "", "Peer ID to connect to")
	flag.BoolVar(&verbose, "verbose", false, "enable libp2p debug logging")

	flag.Parse()
	if hexPSK == "" {
		flag.Usage()
		log.Fatalln("The psk flag is required")
	}
	if remotePeerID == "" {
		flag.Usage()
		log.Fatalln("The remote flag is required")
	}
	targetID, err := peer.Decode(remotePeerID)
	if err != nil {
		log.Fatalln("Unable to hex decode the remote", err)
	}
	psk, err := hex.DecodeString(hexPSK)
	if err != nil {
		log.Fatalln("Unable to hex decode the PSK", err)
	}
	if len(psk) != 32 {
		log.Fatalln("The PSK should be 32 bytes")
	}

	relayAddrInfo, err := peer.AddrInfoFromString(relay)
	if err != nil {
		log.Fatalln(err)
	}

	if verbose {
		logging.SetDebugLogging()
	}
	libp2pctx := context.Background()
	p2pHost, peerRouting, err := client.CreateLibp2pHost(libp2pctx, 0, false, psk, nil, []peer.AddrInfo{*relayAddrInfo})
	if err != nil {
		panic(err)
	}
	log.Println("Started libp2p host on", p2pHost.Addrs())

	//Force the relayfinder of the autorelay to start
	emitReachabilityChanged, _ := p2pHost.EventBus().Emitter(new(event.EvtLocalReachabilityChanged))
	emitReachabilityChanged.Emit(event.EvtLocalReachabilityChanged{Reachability: network.ReachabilityUnknown})
	for {

		log.Println("Peers:", p2pHost.Peerstore().Peers())

		if err = client.ConnectToPeer(libp2pctx, p2pHost, peerRouting, relayAddrInfo, targetID); err != nil {
			log.Println("Unable to connect to remote:", err)
		} else {
			break
		}

		time.Sleep(time.Second * 10)

	}

	// proxying request here
	proxy := NewProxyService(libp2pctx, p2pHost)
	if err := proxy.Serve("0.0.0.0:1082", targetID); err != nil {
		log.Fatal(err)
	}
}
