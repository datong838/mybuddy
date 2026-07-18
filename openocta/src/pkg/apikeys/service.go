package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("api key not found")
	ErrInvalidSecret     = errors.New("invalid api key")
	ErrSecretUnavailable = errors.New("api key secret unavailable")
	ErrPathDenied        = errors.New("path not allowed for this api key")
	ErrModelDenied       = errors.New("model not allowed for this api key")
	ErrTokenLimit        = errors.New("monthly token limit exceeded")
)

// Service manages API keys in a local JSON store.
type Service struct {
	mu        sync.Mutex
	storePath string
}

// NewService creates an API key service backed by storePath.
func NewService(storePath string) (*Service, error) {
	if strings.TrimSpace(storePath) == "" {
		return nil, fmt.Errorf("api key store path required")
	}
	return &Service{storePath: storePath}, nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func generateSecret() (string, string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	secret := "octa_" + hex.EncodeToString(buf)
	prefix := secret
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	return secret, prefix, nil
}

func currentUsageMonth() string {
	return time.Now().UTC().Format("2006-01")
}

func normalizePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		p = normalizePath(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

func normalizeStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func (r *Record) resetUsageIfNeeded() {
	month := currentUsageMonth()
	if r.UsageMonth != month {
		r.UsageMonth = month
		r.MonthlyTokensUsed = 0
	}
}

func toPublic(r Record) PublicEntry {
	return PublicEntry{
		ID:                r.ID,
		Name:              r.Name,
		KeyPrefix:         r.KeyPrefix,
		AllowedPaths:      append([]string(nil), r.AllowedPaths...),
		BindingMode:       r.BindingMode,
		AllowedModels:     append([]string(nil), r.AllowedModels...),
		SkillKeys:         append([]string(nil), r.SkillKeys...),
		McpServers:        append([]string(nil), r.McpServers...),
		DigitalEmployeeID: r.DigitalEmployeeID,
		MonthlyTokenLimit: r.MonthlyTokenLimit,
		MonthlyTokensUsed: r.MonthlyTokensUsed,
		UsageMonth:        r.UsageMonth,
		Enabled:           r.Enabled,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (s *Service) withStore(fn func(*storeFile) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, err := LoadStore(s.storePath)
	if err != nil {
		return err
	}
	if err := fn(store); err != nil {
		return err
	}
	return SaveStore(s.storePath, store)
}

// List returns all API keys (without secrets).
func (s *Service) List() ([]PublicEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	store, err := LoadStore(s.storePath)
	if err != nil {
		return nil, err
	}
	out := make([]PublicEntry, 0, len(store.Keys))
	for _, k := range store.Keys {
		k.resetUsageIfNeeded()
		out = append(out, toPublic(k))
	}
	return out, nil
}

// Create adds a new API key and returns the plaintext secret once.
func (s *Service) Create(p CreateParams) (CreateResult, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return CreateResult{}, fmt.Errorf("name required")
	}
	bindingMode := strings.TrimSpace(p.BindingMode)
	if bindingMode == "" {
		bindingMode = BindingModeResources
	}
	if bindingMode != BindingModeResources && bindingMode != BindingModeEmployee {
		return CreateResult{}, fmt.Errorf("invalid binding mode")
	}
	paths := normalizePaths(p.AllowedPaths)
	if len(paths) == 0 {
		paths = append([]string(nil), DefaultAllowedPaths...)
	}
	secret, prefix, err := generateSecret()
	if err != nil {
		return CreateResult{}, err
	}
	keyEnc, err := encryptSecret(s.storePath, secret)
	if err != nil {
		return CreateResult{}, err
	}
	now := time.Now().UnixMilli()
	rec := Record{
		ID:                uuid.NewString(),
		Name:              name,
		KeyPrefix:         prefix,
		KeyHash:           hashSecret(secret),
		KeyEnc:            keyEnc,
		AllowedPaths:      paths,
		BindingMode:       bindingMode,
		AllowedModels:     normalizeStringList(p.AllowedModels),
		SkillKeys:         normalizeStringList(p.SkillKeys),
		McpServers:        normalizeStringList(p.McpServers),
		DigitalEmployeeID: strings.TrimSpace(p.DigitalEmployeeID),
		MonthlyTokenLimit: p.MonthlyTokenLimit,
		UsageMonth:        currentUsageMonth(),
		Enabled:           true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.withStore(func(store *storeFile) error {
		store.Keys = append(store.Keys, rec)
		return nil
	}); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Entry: toPublic(rec), Secret: secret}, nil
}

// Update patches an existing API key.
func (s *Service) Update(id string, p UpdateParams) (PublicEntry, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return PublicEntry{}, fmt.Errorf("id required")
	}
	var updated PublicEntry
	err := s.withStore(func(store *storeFile) error {
		for i := range store.Keys {
			if store.Keys[i].ID != id {
				continue
			}
			rec := &store.Keys[i]
			if p.Name != nil {
				name := strings.TrimSpace(*p.Name)
				if name == "" {
					return fmt.Errorf("name required")
				}
				rec.Name = name
			}
			if p.AllowedPaths != nil {
				rec.AllowedPaths = normalizePaths(*p.AllowedPaths)
			}
			if p.BindingMode != nil {
				mode := strings.TrimSpace(*p.BindingMode)
				if mode != BindingModeResources && mode != BindingModeEmployee {
					return fmt.Errorf("invalid binding mode")
				}
				rec.BindingMode = mode
			}
			if p.AllowedModels != nil {
				rec.AllowedModels = normalizeStringList(*p.AllowedModels)
			}
			if p.SkillKeys != nil {
				rec.SkillKeys = normalizeStringList(*p.SkillKeys)
			}
			if p.McpServers != nil {
				rec.McpServers = normalizeStringList(*p.McpServers)
			}
			if p.DigitalEmployeeID != nil {
				rec.DigitalEmployeeID = strings.TrimSpace(*p.DigitalEmployeeID)
			}
			if p.MonthlyTokenLimit != nil {
				rec.MonthlyTokenLimit = p.MonthlyTokenLimit
			}
			if p.Enabled != nil {
				rec.Enabled = *p.Enabled
			}
			rec.resetUsageIfNeeded()
			rec.UpdatedAt = time.Now().UnixMilli()
			updated = toPublic(*rec)
			return nil
		}
		return ErrNotFound
	})
	return updated, err
}

// GetSecret returns the stored plaintext secret for admin UI.
func (s *Service) GetSecret(id string) (SecretResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return SecretResult{}, fmt.Errorf("id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	store, err := LoadStore(s.storePath)
	if err != nil {
		return SecretResult{}, err
	}
	for _, rec := range store.Keys {
		if rec.ID != id {
			continue
		}
		if strings.TrimSpace(rec.KeyEnc) == "" {
			return SecretResult{}, ErrSecretUnavailable
		}
		secret, err := decryptSecret(s.storePath, rec.KeyEnc)
		if err != nil {
			return SecretResult{}, err
		}
		return SecretResult{ID: rec.ID, Secret: secret}, nil
	}
	return SecretResult{}, ErrNotFound
}

// RegenerateSecret rotates the secret and returns the new plaintext value.
func (s *Service) RegenerateSecret(id string) (RegenerateResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RegenerateResult{}, fmt.Errorf("id required")
	}
	var result RegenerateResult
	err := s.withStore(func(store *storeFile) error {
		for i := range store.Keys {
			if store.Keys[i].ID != id {
				continue
			}
			rec := &store.Keys[i]
			secret, prefix, err := generateSecret()
			if err != nil {
				return err
			}
			keyEnc, err := encryptSecret(s.storePath, secret)
			if err != nil {
				return err
			}
			rec.KeyPrefix = prefix
			rec.KeyHash = hashSecret(secret)
			rec.KeyEnc = keyEnc
			rec.UpdatedAt = time.Now().UnixMilli()
			result = RegenerateResult{Entry: toPublic(*rec), Secret: secret}
			return nil
		}
		return ErrNotFound
	})
	return result, err
}

// Remove deletes an API key by id.
func (s *Service) Remove(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id required")
	}
	return s.withStore(func(store *storeFile) error {
		for i, k := range store.Keys {
			if k.ID == id {
				store.Keys = append(store.Keys[:i], store.Keys[i+1:]...)
				return nil
			}
		}
		return ErrNotFound
	})
}

// Authenticate validates a bearer token and returns the matching record.
func (s *Service) Authenticate(secret string) (*Record, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrInvalidSecret
	}
	hash := hashSecret(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	store, err := LoadStore(s.storePath)
	if err != nil {
		return nil, err
	}
	for i := range store.Keys {
		rec := &store.Keys[i]
		if rec.KeyHash != hash {
			continue
		}
		if !rec.Enabled {
			return nil, ErrInvalidSecret
		}
		rec.resetUsageIfNeeded()
		cp := *rec
		return &cp, nil
	}
	return nil, ErrInvalidSecret
}

// PathAllowed checks whether requestPath matches any allowed prefix.
func PathAllowed(rec *Record, requestPath string) bool {
	if rec == nil {
		return false
	}
	path := normalizePath(requestPath)
	for _, allowed := range rec.AllowedPaths {
		allowed = normalizePath(allowed)
		if allowed == "" {
			continue
		}
		if path == allowed || strings.HasPrefix(path, allowed+"/") {
			return true
		}
	}
	return false
}

// ModelAllowed checks model against whitelist (empty whitelist = allow all).
func ModelAllowed(rec *Record, model string) bool {
	if rec == nil {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	if len(rec.AllowedModels) == 0 {
		return true
	}
	lower := strings.ToLower(model)
	for _, allowed := range rec.AllowedModels {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if strings.EqualFold(allowed, model) {
			return true
		}
		// Also match bare model id when whitelist uses provider/model.
		if strings.Contains(allowed, "/") {
			parts := strings.Split(allowed, "/")
			if len(parts) >= 2 && strings.EqualFold(parts[len(parts)-1], model) {
				return true
			}
		}
		if strings.EqualFold(allowed, lower) {
			return true
		}
	}
	return false
}

// CheckTokenBudget returns ErrTokenLimit when monthly limit exceeded.
func CheckTokenBudget(rec *Record, additional int64) error {
	if rec == nil || rec.MonthlyTokenLimit == nil || *rec.MonthlyTokenLimit <= 0 {
		return nil
	}
	rec.resetUsageIfNeeded()
	if rec.MonthlyTokensUsed+additional > *rec.MonthlyTokenLimit {
		return ErrTokenLimit
	}
	return nil
}

// RecordUsage increments monthly token usage for a key id.
func (s *Service) RecordUsage(id string, tokens int64) error {
	if tokens <= 0 {
		return nil
	}
	return s.withStore(func(store *storeFile) error {
		for i := range store.Keys {
			if store.Keys[i].ID != id {
				continue
			}
			rec := &store.Keys[i]
			rec.resetUsageIfNeeded()
			rec.MonthlyTokensUsed += tokens
			rec.UpdatedAt = time.Now().UnixMilli()
			return nil
		}
		return ErrNotFound
	})
}
