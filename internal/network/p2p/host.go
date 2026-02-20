// Package p2p provides the libp2p-based peer-to-peer networking layer.
package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/rs/zerolog"
)

const (
	// ConcordProtocol is the libp2p protocol ID for Concord messages.
	ConcordProtocol = protocol.ID("/concord/1.0.0")

	// MDNSServiceTag is the mDNS service tag for LAN discovery.
	MDNSServiceTag = "concord.local"

	// DefaultPort is the default libp2p listen port.
	DefaultPort = 0 // random port
)

// Config holds the P2P host configuration.
type Config struct {
	ListenPort    int
	EnableMDNS    bool // LAN peer discovery
	EnableDHT     bool // Internet peer discovery
	BootstrapPeers []string
}

// DefaultConfig returns a sensible default P2P configuration.
func DefaultConfig() Config {
	return Config{
		ListenPort: DefaultPort,
		EnableMDNS: true,
		EnableDHT:  true,
	}
}

// PeerInfo holds information about a connected peer.
type PeerInfo struct {
	ID        string   `json:"id"`
	Addresses []string `json:"addresses"`
	Connected bool     `json:"connected"`
}

// MessageHandler is called when a message is received from a peer.
type MessageHandler func(peerID string, data []byte)

// Host wraps a libp2p host with Concord-specific functionality.
type Host struct {
	mu       sync.RWMutex
	host     host.Host
	dht      *dht.IpfsDHT
	mdns     mdns.Service
	handler  MessageHandler
	logger   zerolog.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates and starts a new P2P host.
func New(cfg Config, logger zerolog.Logger) (*Host, error) {
	ctx, cancel := context.WithCancel(context.Background())

	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.ListenPort)
	quicAddr := fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", cfg.ListenPort)

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddr, quicAddr),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelay(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("p2p: create host: %w", err)
	}

	p2pHost := &Host{
		host:   h,
		logger: logger.With().Str("component", "p2p").Logger(),
		ctx:    ctx,
		cancel: cancel,
	}

	// Set stream handler for incoming messages
	h.SetStreamHandler(ConcordProtocol, p2pHost.handleStream)

	logger.Info().
		Str("peer_id", h.ID().String()).
		Strs("addrs", multiaddrsToStrings(h)).
		Msg("P2P host started")

	// Start discovery
	if cfg.EnableMDNS {
		if err := p2pHost.startMDNS(); err != nil {
			logger.Warn().Err(err).Msg("mDNS discovery failed to start")
		}
	}

	if cfg.EnableDHT {
		if err := p2pHost.startDHT(ctx, cfg.BootstrapPeers); err != nil {
			logger.Warn().Err(err).Msg("DHT discovery failed to start")
		}
	}

	return p2pHost, nil
}

// ID returns the host's peer ID.
func (h *Host) ID() string {
	return h.host.ID().String()
}

// Addrs returns the host's listen addresses.
func (h *Host) Addrs() []string {
	return multiaddrsToStrings(h.host)
}

// OnMessage registers a handler for incoming messages.
func (h *Host) OnMessage(handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handler = handler
}

// Connect connects to a peer by their multiaddr string.
func (h *Host) Connect(ctx context.Context, addrStr string) error {
	addr, err := peer.AddrInfoFromString(addrStr)
	if err != nil {
		return fmt.Errorf("p2p: parse addr: %w", err)
	}

	if err := h.host.Connect(ctx, *addr); err != nil {
		return fmt.Errorf("p2p: connect to %s: %w", addr.ID, err)
	}

	h.logger.Info().
		Str("peer_id", addr.ID.String()).
		Msg("connected to peer")

	return nil
}

// ConnectPeer connects to a peer by their AddrInfo.
func (h *Host) ConnectPeer(ctx context.Context, info peer.AddrInfo) error {
	if err := h.host.Connect(ctx, info); err != nil {
		return fmt.Errorf("p2p: connect to %s: %w", info.ID, err)
	}

	h.logger.Info().
		Str("peer_id", info.ID.String()).
		Msg("connected to peer")

	return nil
}

// SendData sends data to a specific peer.
func (h *Host) SendData(ctx context.Context, peerIDStr string, data []byte) error {
	pid, err := peer.Decode(peerIDStr)
	if err != nil {
		return fmt.Errorf("p2p: decode peer id: %w", err)
	}

	stream, err := h.host.NewStream(ctx, pid, ConcordProtocol)
	if err != nil {
		return fmt.Errorf("p2p: open stream to %s: %w", peerIDStr, err)
	}
	defer stream.Close()

	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("p2p: write to %s: %w", peerIDStr, err)
	}

	return nil
}

// Peers returns info about all connected peers.
func (h *Host) Peers() []PeerInfo {
	conns := h.host.Network().Conns()
	peers := make([]PeerInfo, 0, len(conns))
	seen := make(map[peer.ID]bool)

	for _, conn := range conns {
		pid := conn.RemotePeer()
		if seen[pid] {
			continue
		}
		seen[pid] = true

		addrs := make([]string, 0)
		for _, addr := range h.host.Peerstore().Addrs(pid) {
			addrs = append(addrs, addr.String())
		}

		peers = append(peers, PeerInfo{
			ID:        pid.String(),
			Addresses: addrs,
			Connected: h.host.Network().Connectedness(pid) == network.Connected,
		})
	}

	return peers
}

// PeerCount returns the number of connected peers.
func (h *Host) PeerCount() int {
	conns := h.host.Network().Conns()
	seen := make(map[peer.ID]bool)
	for _, conn := range conns {
		seen[conn.RemotePeer()] = true
	}
	return len(seen)
}

// Stop shuts down the P2P host.
func (h *Host) Stop() error {
	h.cancel()

	if h.mdns != nil {
		if err := h.mdns.Close(); err != nil {
			h.logger.Warn().Err(err).Msg("failed to close mDNS")
		}
	}

	if h.dht != nil {
		if err := h.dht.Close(); err != nil {
			h.logger.Warn().Err(err).Msg("failed to close DHT")
		}
	}

	if err := h.host.Close(); err != nil {
		return fmt.Errorf("p2p: close host: %w", err)
	}

	h.logger.Info().Msg("P2P host stopped")
	return nil
}

// handleStream processes incoming libp2p streams.
func (h *Host) handleStream(s network.Stream) {
	defer s.Close()

	h.mu.RLock()
	handler := h.handler
	h.mu.RUnlock()

	if handler == nil {
		return
	}

	buf := make([]byte, 64*1024) // 64KB read buffer
	n, err := s.Read(buf)
	if err != nil {
		h.logger.Debug().Err(err).
			Str("from", s.Conn().RemotePeer().String()).
			Msg("stream read error")
		return
	}

	handler(s.Conn().RemotePeer().String(), buf[:n])
}

// startMDNS sets up mDNS for LAN peer discovery.
func (h *Host) startMDNS() error {
	notifee := &mdnsNotifee{host: h}
	svc := mdns.NewMdnsService(h.host, MDNSServiceTag, notifee)
	if err := svc.Start(); err != nil {
		return err
	}
	h.mdns = svc
	h.logger.Info().Msg("mDNS discovery started")
	return nil
}

// startDHT sets up the Kademlia DHT for internet peer discovery.
func (h *Host) startDHT(ctx context.Context, bootstrapPeers []string) error {
	kadDHT, err := dht.New(ctx, h.host, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		return fmt.Errorf("p2p: create DHT: %w", err)
	}

	if err := kadDHT.Bootstrap(ctx); err != nil {
		return fmt.Errorf("p2p: bootstrap DHT: %w", err)
	}

	h.dht = kadDHT

	// Connect to bootstrap peers
	for _, addrStr := range bootstrapPeers {
		addr, err := peer.AddrInfoFromString(addrStr)
		if err != nil {
			h.logger.Warn().Str("addr", addrStr).Err(err).Msg("invalid bootstrap peer")
			continue
		}
		go func(ai peer.AddrInfo) {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := h.host.Connect(ctx, ai); err != nil {
				h.logger.Debug().Str("peer", ai.ID.String()).Err(err).Msg("bootstrap connect failed")
			}
		}(*addr)
	}

	h.logger.Info().Msg("DHT discovery started")
	return nil
}

// FindPeers uses the DHT to discover peers advertising a rendezvous string.
func (h *Host) FindPeers(ctx context.Context, rendezvous string) (<-chan peer.AddrInfo, error) {
	if h.dht == nil {
		return nil, fmt.Errorf("p2p: DHT not initialized")
	}

	routingDiscovery := drouting.NewRoutingDiscovery(h.dht)

	// Advertise ourselves
	_, err := routingDiscovery.Advertise(ctx, rendezvous)
	if err != nil {
		return nil, fmt.Errorf("p2p: advertise: %w", err)
	}

	// Find others
	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		return nil, fmt.Errorf("p2p: find peers: %w", err)
	}

	return peerChan, nil
}

// LibP2PHost returns the underlying libp2p host (for advanced use).
func (h *Host) LibP2PHost() host.Host {
	return h.host
}

// mdnsNotifee handles mDNS peer discovery events.
type mdnsNotifee struct {
	host *Host
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.host.host.ID() {
		return // ignore self
	}

	n.host.logger.Info().
		Str("peer_id", pi.ID.String()).
		Msg("mDNS: peer discovered")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := n.host.host.Connect(ctx, pi); err != nil {
		n.host.logger.Debug().Err(err).
			Str("peer_id", pi.ID.String()).
			Msg("mDNS: auto-connect failed")
	}
}

func multiaddrsToStrings(h host.Host) []string {
	addrs := h.Addrs()
	result := make([]string, len(addrs))
	for i, a := range addrs {
		result[i] = fmt.Sprintf("%s/p2p/%s", a, h.ID())
	}
	return result
}
