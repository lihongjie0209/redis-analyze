package models

// KeyInfo holds information about a single Redis key.
type KeyInfo struct {
	Key      string `json:"key"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`     // bytes from MEMORY USAGE, -1 if unavailable
	Encoding string `json:"encoding"` // internal encoding
	IdleTime int64  `json:"idle_time"` // seconds since last access
}

// TypeStats groups statistics by Redis data type.
type TypeStats struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Total int64  `json:"total_bytes"`
	Avg   int64  `json:"avg_bytes"`
	Min   int64  `json:"min_bytes"`
	Max   int64  `json:"max_bytes"`
}

// PrefixStats groups statistics by key prefix / namespace.
type PrefixStats struct {
	Prefix string           `json:"prefix"`
	Count  int64            `json:"count"`
	Total  int64            `json:"total_bytes"`
	Types  map[string]int64 `json:"types"` // type -> count
}

// SummaryStats contains overall summary of the analysis.
type SummaryStats struct {
	TotalKeys   int64 `json:"total_keys"`
	TotalMemory int64 `json:"total_memory_bytes"`
	AvgKeySize  int64 `json:"avg_key_size_bytes"`
	MinKeySize  int64 `json:"min_key_size_bytes"`
	MaxKeySize  int64 `json:"max_key_size_bytes"`
}

// Report is the complete analysis result.
type Report struct {
	Summary      SummaryStats  `json:"summary"`
	ByType       []TypeStats   `json:"by_type"`
	ByPrefix     []PrefixStats `json:"by_prefix"`
	TopKeys      []KeyInfo     `json:"top_keys"`
	ScannedCount int64         `json:"scanned_count"`
	ScanDuration int64         `json:"scan_duration_ms"`
	ServerInfo   ServerInfo    `json:"server_info"`
	Prefixes     []string      `json:"prefixes"`
}

// ServerInfo holds basic Redis server info.
type ServerInfo struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Mode      string `json:"mode"`
	DB        int    `json:"db"`
	Version   string `json:"version"`
	Uptime    int64  `json:"uptime_seconds"`
	UsedMem   int64  `json:"used_memory"`
	Os        string `json:"os"`
	KeysCount int64  `json:"keys_count"` // from DBSIZE or INFO keyspace
}

// ScanOptions holds all scanning configuration.
type ScanOptions struct {
	Host            string
	Port            int
	Password        string
	Username        string
	DB              int
	Mode            string // standalone, cluster, sentinel
	Addrs           []string
	MasterName      string
	SentinelPass    string
	Prefixes        []string
	TLS             bool
	Samples         int
	TopN            int
	Concurrency     int
	Timeout         int // seconds
	Separator       string
	PrefixDepth     int
	NoProgress      bool
	Format          string // table, json, csv
	ReportInterval  int    // seconds between intermediate reports (0 = disabled)
	BatchSize       int    // keys per batch (0 = auto, default 50)
	ScanMode        string // scanning strategy: auto, lua, pipeline, sequential
}
