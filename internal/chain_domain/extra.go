package chain_domain

import "time"

// Protocol limits make hostile inputs explicit and reviewable.
const (
	MaxValidators      = 128
	MaxTxPerBlock      = 4096
	MaxKeyBytes        = 256
	MaxValueBytes      = 4096
	MaxPeers           = 1024
	MaxSnapshotEntries = 1000000
)

type ChainConfig struct {
	Version         uint32
	BlockTime       time.Duration
	MaxBlockBytes   int
	MaxGas          uint64
	MinFee          uint64
	EpochLength     Height
	EnableProofs    bool
	EnableSnapshots bool
}

func DefaultChainConfig() ChainConfig {
	return ChainConfig{Version: 1, BlockTime: 2 * time.Second, MaxBlockBytes: 2 << 20, MaxGas: 10000000, MinFee: 1, EpochLength: 100, EnableProofs: true, EnableSnapshots: true}
}

type Account struct {
	Address     string
	Balance     uint64
	Nonce       uint64
	CodeHash    string
	StorageRoot string
	Frozen      bool
	UpdatedAt   time.Time
}

type StateRoot struct {
	Height     Height
	Hash       string
	Algorithm  string
	Version    uint32
	EntryCount uint64
	CreatedAt  time.Time
}

type Checkpoint struct {
	Height     Height
	BlockHash  string
	StateRoot  string
	Validators []Validator
	Signature  string
	CreatedAt  time.Time
}

type PeerRecord struct {
	ID           string
	Address      string
	Version      string
	Capabilities []string
	Score        int
	Failures     uint32
	ConnectedAt  time.Time
	LastSeen     time.Time
	BannedUntil  time.Time
}

type SyncPlan struct {
	PeerID             string
	From               Height
	To                 Height
	ChunkSize          uint64
	MaxBytes           uint64
	VerifyHeaders      bool
	VerifyTransactions bool
	VerifyStateRoots   bool
}

type RollbackPlan struct {
	Target        Height
	Current       Height
	Reason        string
	RequiredVotes uint64
	PreparedBy    string
	PreparedAt    time.Time
	Approved      bool
}

type UpgradeMarker struct {
	FromVersion      uint32
	ToVersion        uint32
	ActivationHeight Height
	Digest           string
	Migrated         bool
	AppliedAt        time.Time
}

type KeyRotation struct {
	Validator       string
	OldKey          string
	NewKey          string
	EffectiveHeight Height
	Proof           string
	Revoked         bool
}

type AuditRecord struct {
	Sequence     uint64
	Action       string
	Actor        string
	Resource     string
	PayloadHash  string
	PreviousHash string
	Hash         string
	At           time.Time
}

func (c ChainConfig) Valid() bool {
	if c.Version == 0 {
		return false
	}
	if c.BlockTime <= 0 {
		return false
	}
	if c.MaxBlockBytes <= 0 || c.MaxBlockBytes > 64<<20 {
		return false
	}
	if c.MaxGas == 0 || c.EpochLength == 0 {
		return false
	}
	return true
}

func (a Account) Valid() bool {
	if a.Address == "" || len(a.Address) > 128 {
		return false
	}
	if len(a.CodeHash) > 128 || len(a.StorageRoot) > 128 {
		return false
	}
	return true
}

func (p PeerRecord) IsBanned(now time.Time) bool {
	return !p.BannedUntil.IsZero() && p.BannedUntil.After(now)
}

func (p PeerRecord) Healthy() bool {
	return p.Score >= -10 && p.Failures < 5 && !p.IsBanned(time.Now())
}

func (s SyncPlan) Valid() bool {
	if s.PeerID == "" || s.To < s.From {
		return false
	}
	if s.ChunkSize == 0 || s.ChunkSize > 10000 {
		return false
	}
	if s.MaxBytes == 0 {
		return false
	}
	return true
}

func (r RollbackPlan) Valid() bool {
	return r.Target < r.Current && r.Reason != "" && r.RequiredVotes > 0
}

func (m UpgradeMarker) Ready(h Height) bool {
	return m.FromVersion > 0 && m.ToVersion > m.FromVersion && h >= m.ActivationHeight
}

func (k KeyRotation) Ready(h Height) bool {
	return k.Validator != "" && k.OldKey != "" && k.NewKey != "" && h >= k.EffectiveHeight && !k.Revoked
}

func (a AuditRecord) Valid() bool {
	return a.Sequence > 0 && a.Action != "" && a.Hash != "" && a.At.IsZero() == false
}

// Event names are stable API values used by subscribers.
const (
	EventBlockProposed    = "block.proposed"
	EventBlockCommitted   = "block.committed"
	EventBlockRejected    = "block.rejected"
	EventTxAccepted       = "transaction.accepted"
	EventTxRejected       = "transaction.rejected"
	EventPeerConnected    = "peer.connected"
	EventPeerBanned       = "peer.banned"
	EventSnapshotCreated  = "snapshot.created"
	EventSnapshotVerified = "snapshot.verified"
	EventRollbackPrepared = "rollback.prepared"
	EventUpgradeActivated = "upgrade.activated"
	EventKeyRotated       = "key.rotated"
)

type Event struct {
	Name   string
	Height Height
	Data   map[string]string
	At     time.Time
}

func NewEvent(name string, h Height) Event {
	return Event{Name: name, Height: h, Data: map[string]string{}, At: time.Now().UTC()}
}

func (e *Event) Put(k, v string) {
	if e.Data == nil {
		e.Data = map[string]string{}
	}
	e.Data[k] = v
}

func (e Event) Get(k string) string { return e.Data[k] }

func (e Event) Valid() bool { return e.Name != "" && !e.At.IsZero() }
