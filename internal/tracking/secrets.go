package tracking

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/99designs/keyring"
	"github.com/salmonumbrella/fastmail-cli/internal/keyringutil"
)

const keyringService = "email-tracking"

var (
	errMissingTrackingKey = errors.New("missing tracking key")
	errMissingAdminKey    = errors.New("missing admin key")
)

const (
	legacyTrackingKeySecretKey         = "tracking_key"                 // #nosec G101
	adminKeySecretKey                  = "admin_key"                    // #nosec G101
	trackingKeyCurrentVersionSecretKey = "tracking_key_current_version" // #nosec G101
	trackingKeyVersionSecretKeyPrefix  = "tracking_key_v"
)

const defaultTrackingKeyVersionInt = 1

func openKeyring() (keyring.Keyring, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	ring, err := keyring.Open(keyring.Config{
		ServiceName: keyringService,
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.WinCredBackend,
			keyring.SecretServiceBackend,
			keyring.FileBackend,
		},
		FileDir:          configDir,
		FilePasswordFunc: keyring.TerminalPrompt,
	})
	if err != nil {
		return nil, err
	}
	return keyringutil.Wrap(ring, keyringutil.DefaultTimeout), nil
}

// SaveSecrets stores a v1 tracking key in the keyring.
func SaveSecrets(trackingKey, adminKey string) error {
	return SaveTrackingKeys(map[int]string{defaultTrackingKeyVersionInt: trackingKey}, adminKey, defaultTrackingKeyVersionInt)
}

// SaveTrackingKeys stores tracking keys by version in the keyring.
func SaveTrackingKeys(trackingKeys map[int]string, adminKey string, currentVersion int) error {
	if adminKey == "" {
		return errMissingAdminKey
	}

	currentVersion = normalizeTrackingVersion(currentVersion)
	if currentVersion <= 0 {
		return fmt.Errorf("invalid current tracking version: %d", currentVersion)
	}

	currentKey, ok := trackingKeys[currentVersion]
	if !ok || strings.TrimSpace(currentKey) == "" {
		return errMissingTrackingKey
	}

	ring, err := openKeyring()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	versions := sortedTrackingVersionsFromMap(trackingKeys)
	if len(versions) == 0 {
		return errMissingTrackingKey
	}

	for _, version := range versions {
		key, ok := trackingKeys[version]
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		if err := ring.Set(keyring.Item{
			Key:  trackingKeyName(version),
			Data: []byte(key),
		}); err != nil {
			return fmt.Errorf("store tracking key v%d: %w", version, err)
		}
	}

	if err := ring.Set(keyring.Item{
		Key:  legacyTrackingKeySecretKey,
		Data: []byte(currentKey),
	}); err != nil {
		return fmt.Errorf("store tracking key: %w", err)
	}

	if err := ring.Set(keyring.Item{
		Key:  trackingKeyCurrentVersionSecretKey,
		Data: []byte(strconv.Itoa(currentVersion)),
	}); err != nil {
		return fmt.Errorf("store current tracking version: %w", err)
	}

	if err := ring.Set(keyring.Item{
		Key:  adminKeySecretKey,
		Data: []byte(adminKey),
	}); err != nil {
		return fmt.Errorf("store admin key: %w", err)
	}

	return nil
}

// LoadSecrets retrieves the current tracking key and admin key from the keyring.
func LoadSecrets(versions []int, currentVersion int) (trackingKey, adminKey string, err error) {
	keys, adminKey, err := LoadTrackingKeys(versions, currentVersion)
	if err != nil {
		return "", "", err
	}
	currentVersion = normalizeTrackingVersion(currentVersion)
	if key, ok := keys[currentVersion]; ok {
		trackingKey = key
		return trackingKey, adminKey, nil
	}

	orderedVersions := sortedTrackingVersionsFromMap(keys)
	for i := len(orderedVersions) - 1; i >= 0; i-- {
		if key := keys[orderedVersions[i]]; strings.TrimSpace(key) != "" {
			trackingKey = key
			break
		}
	}
	return trackingKey, adminKey, nil
}

// LoadTrackingKeys retrieves tracking keys by version from the keyring.
func LoadTrackingKeys(versions []int, currentVersion int) (trackingKeys map[int]string, adminKey string, err error) {
	ring, err := openKeyring()
	if err != nil {
		// If keyring unavailable, return empty (config might have keys inline)
		if os.IsNotExist(err) {
			return map[int]string{}, "", nil
		}
		return nil, "", fmt.Errorf("open keyring: %w", err)
	}

	currentVersion = normalizeTrackingVersion(currentVersion)
	if len(versions) == 0 {
		versions = []int{currentVersion}
	}
	versions = sortedTrackingVersions(versions)
	if !containsTrackingVersion(versions, currentVersion) {
		versions = append([]int{currentVersion}, versions...)
	}
	trackingKeys = map[int]string{}

	for _, version := range versions {
		key, loadErr := loadTrackingKeyForVersion(ring, version)
		if loadErr != nil {
			return nil, "", loadErr
		}
		if key != "" {
			trackingKeys[version] = key
		}
	}

	if currentVersion == defaultTrackingKeyVersionInt {
		if _, ok := trackingKeys[currentVersion]; !ok {
			legacyItem, legacyErr := ring.Get(legacyTrackingKeySecretKey)
			if legacyErr == nil {
				trackingKeys[currentVersion] = strings.TrimSpace(string(legacyItem.Data))
			} else if !errors.Is(legacyErr, keyring.ErrKeyNotFound) {
				return nil, "", fmt.Errorf("read legacy tracking key: %w", legacyErr)
			}
		}
	}

	if currentItem, currentVersionErr := ring.Get(trackingKeyCurrentVersionSecretKey); currentVersionErr == nil {
		if parsedVersion, parseErr := strconv.Atoi(strings.TrimSpace(string(currentItem.Data))); parseErr == nil {
			parsedVersion = normalizeTrackingVersion(parsedVersion)
			if parsedVersion != currentVersion {
				currentVersion = parsedVersion
				if !containsTrackingVersion(versions, currentVersion) {
					currentItemKey, loadCurrentErr := loadTrackingKeyForVersion(ring, currentVersion)
					if loadCurrentErr != nil {
						return nil, "", loadCurrentErr
					}
					if currentItemKey != "" {
						trackingKeys[currentVersion] = currentItemKey
					}
				}
			}
		}
	} else if !errors.Is(currentVersionErr, keyring.ErrKeyNotFound) {
		return nil, "", fmt.Errorf("read current tracking version: %w", currentVersionErr)
	}

	akItem, err := ring.Get(adminKeySecretKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return trackingKeys, "", nil
		}
		return nil, "", fmt.Errorf("read admin key: %w", err)
	}

	adminKey = string(akItem.Data)
	return trackingKeys, strings.TrimSpace(adminKey), nil
}

func trackingKeyName(version int) string {
	return fmt.Sprintf("%s%d", trackingKeyVersionSecretKeyPrefix, version)
}

func loadTrackingKeyForVersion(ring keyring.Keyring, version int) (string, error) {
	keyName := trackingKeyName(version)
	item, err := ring.Get(keyName)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("read tracking key %d: %w", version, err)
	}
	return strings.TrimSpace(string(item.Data)), nil
}

func normalizeTrackingVersion(version int) int {
	if version <= 0 {
		return defaultTrackingKeyVersionInt
	}
	return version
}

func sortedTrackingVersions(versions []int) []int {
	out := make([]int, 0, len(versions))
	seen := map[int]struct{}{}
	for _, version := range versions {
		if version <= 0 {
			continue
		}
		if _, ok := seen[version]; ok {
			continue
		}
		seen[version] = struct{}{}
		out = append(out, version)
	}
	sort.Ints(out)
	return out
}

func sortedTrackingVersionsFromMap(trackingKeys map[int]string) []int {
	versions := make([]int, 0, len(trackingKeys))
	for version := range trackingKeys {
		if version > 0 {
			versions = append(versions, version)
		}
	}
	sort.Ints(versions)
	return versions
}

func containsTrackingVersion(versions []int, version int) bool {
	for _, existing := range versions {
		if existing == version {
			return true
		}
	}
	return false
}
