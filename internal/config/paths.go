package config

const (
	AppName = "fastmail-cli"

	// CredentialsDirEnvVarName controls the credential storage root directory.
	// fastmail-cli keyring files are stored under: <dir>/fastmail-cli/keyring.
	CredentialsDirEnvVarName = "FASTMAIL_CREDENTIALS_DIR" // #nosec G101 -- environment variable name

	// SharedCredentialsDirEnvVarName is a shared OpenClaw-compatible credential
	// root used when FASTMAIL_CREDENTIALS_DIR is unset.
	SharedCredentialsDirEnvVarName = "OPENCLAW_CREDENTIALS_DIR" // #nosec G101 -- environment variable name

	// KeyringPasswordEnvVarName provides the keyring file-backend password for
	// non-interactive environments.
	KeyringPasswordEnvVarName = "FASTMAIL_KEYRING_PASSWORD" // #nosec G101 -- environment variable name

	// KeyringBackendEnvVarName controls keyring backend selection. Supported
	// values: auto|default|file|keychain|wincred|secret-service.
	KeyringBackendEnvVarName = "FASTMAIL_KEYRING_BACKEND"
)
