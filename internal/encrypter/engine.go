// Package encrypter implements the core cryptographic operations for
// PersonalSecureEncrypter using industry-standard algorithms.
//
// # Cryptographic Design
//
// Key Derivation: Argon2id (RFC 9106)
//   - The RECOMMENDED password hashing function by OWASP and IETF.
//   - Argon2id combines Argon2i (side-channel resistant) and Argon2d
//     (GPU-resistant) for the best overall security.
//   - Parameters: 3 iterations, 64 MB memory, 4 threads, 32-byte key.
//
// Encryption: AES-256-GCM (NIST SP 800-38D)
//   - Authenticated Encryption with Associated Data (AEAD).
//   - Provides both confidentiality AND integrity in a single pass.
//   - 256-bit key ⇒ 128-bit security level.
//   - 12-byte random nonce (standard for GCM).
//   - 16-byte authentication tag appended to ciphertext by GCM.
//
// # Encrypted File Format
//
//	+----------+----------+-----------+---------------------------+
//	|  Header  |   Salt   |   Nonce   |  Ciphertext + GCM Tag     |
//	|  5 bytes | 16 bytes | 12 bytes  |  len(plaintext) + 16 bytes|
//	+----------+----------+-----------+---------------------------+
//
//   - Header: "PSEv1" — allows instant detection of encrypted files.
//   - Salt: unique per file, used for Argon2id key derivation.
//   - Nonce: unique per file, used for AES-GCM encryption.
//   - Ciphertext: AES-256-GCM output with the authentication tag.
//
// # Security Properties
//
//   - Each file gets a unique salt → unique key → nonce reuse is impossible.
//   - Wrong password ⇒ GCM tag verification fails ⇒ error returned.
//   - Tampered ciphertext ⇒ GCM tag verification fails ⇒ error returned.
//   - Key material is zeroed from memory after use (defense-in-depth).
//   - No password, key, or plaintext is ever written to disk as temp files.
package encrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// MagicHeader identifies files encrypted by PersonalSecureEncrypter.
	// Changing this in a future version (e.g. "PSEv2") would allow
	// backward-compatible format detection.
	MagicHeader = "PSEv1"

	// HeaderSize is the byte length of MagicHeader.
	HeaderSize = 5

	// SaltSize is the number of random bytes used for Argon2id.
	// 16 bytes (128 bits) is the OWASP recommended minimum.
	SaltSize = 16

	// NonceSize is the number of random bytes used for AES-GCM.
	// 12 bytes (96 bits) is the NIST standard for GCM nonces.
	NonceSize = 12

	// KeySize is the AES key length in bytes (256 bits).
	KeySize = 32

	// Argon2Time is the number of Argon2id iterations.
	// Higher = slower brute-force, but also slower for the user.
	// 3 is the OWASP recommended minimum for interactive logins.
	Argon2Time = 3

	// Argon2Memory is the Argon2id memory cost in KiB.
	// 64 MB makes GPU attacks extremely expensive.
	Argon2Memory = 64 * 1024

	// Argon2Threads is the Argon2id parallelism factor.
	Argon2Threads = 4

	// MaxFileSize is the maximum file size for in-memory encryption (2 GB).
	// Files larger than this are rejected to prevent out-of-memory crashes.
	MaxFileSize int64 = 2 * 1024 * 1024 * 1024
)

// Engine performs file encryption and decryption operations.
// It holds a reference to the application settings (extension, output folder).
type Engine struct {
	settings *Settings
}

// NewEngine creates a new Engine with the given settings.
func NewEngine(settings *Settings) *Engine {
	return &Engine{settings: settings}
}

// deriveKey uses Argon2id to derive a 256-bit encryption key from a password.
//
// SECURITY:
//   - Argon2id is memory-hard, so brute-force attacks require ~64 MB per guess.
//   - The unique salt ensures two identical passwords produce different keys.
//   - The caller MUST zero the returned key slice after use.
func (e *Engine) deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, Argon2Time, Argon2Memory, Argon2Threads, KeySize)
}

// EncryptFile encrypts a single file and writes the result with the configured
// extension appended (e.g. "photo.jpg" → "photo.jpg.pse").
//
// Returns the output file path on success.
func (e *Engine) EncryptFile(inputPath string, password []byte) (string, error) {
	// ── Validate input path (symlink + size check) ────────────────
	linfo, err := os.Lstat(inputPath)
	if err != nil {
		return "", fmt.Errorf("cannot access source file")
	}
	if linfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("symlinks are not supported for security reasons")
	}
	if linfo.Size() > MaxFileSize {
		return "", fmt.Errorf("file too large (max %d MB)", MaxFileSize/(1024*1024))
	}

	// ── Read plaintext ─────────────────────────────────────────────
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("cannot read source file")
	}

	// ── Generate random salt (16 bytes) ────────────────────────────
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("cryptographic random generation failed")
	}

	// ── Derive key via Argon2id ────────────────────────────────────
	key := e.deriveKey(password, salt)
	defer ClearBytes(key)

	// ── Create AES-256 block cipher ────────────────────────────────
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("cipher initialization failed")
	}

	// ── Create GCM wrapper (authenticated encryption) ──────────────
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM initialization failed")
	}

	// ── Generate random nonce (12 bytes) ───────────────────────────
	// SECURITY: Because we derive a unique key per file (unique salt),
	// even if the same nonce were reused, security would not be
	// compromised. We still use a random nonce for defense-in-depth.
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("cryptographic random generation failed")
	}

	// ── Encrypt ────────────────────────────────────────────────────
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// ── Assemble output: Header ‖ Salt ‖ Nonce ‖ Ciphertext+Tag ───
	outputSize := HeaderSize + SaltSize + NonceSize + len(ciphertext)
	output := make([]byte, 0, outputSize)
	output = append(output, []byte(MagicHeader)...)
	output = append(output, salt...)
	output = append(output, nonce...)
	output = append(output, ciphertext...)

	// ── Determine output path ──────────────────────────────────────
	ext := e.settings.Extension
	if ext == "" {
		ext = ".pse"
	}
	outputPath := inputPath + ext

	if e.settings.OutputFolder != "" {
		outputPath = filepath.Join(e.settings.OutputFolder, filepath.Base(inputPath)+ext)
	}

	// ── Ensure we don't silently overwrite an existing file ────────
	outputPath = safePath(outputPath)

	// ── Write encrypted file with restrictive permissions ──────────
	// 0600 = owner read/write only — no other users can access.
	if err := os.WriteFile(outputPath, output, 0600); err != nil {
		return "", fmt.Errorf("failed to write encrypted file")
	}

	// Zero plaintext from memory.
	ClearBytes(plaintext)

	return outputPath, nil
}

// DecryptFile decrypts a PSE-encrypted file and writes the original content.
// The output filename is derived by stripping the encrypted extension.
//
// Returns the output file path on success.
func (e *Engine) DecryptFile(inputPath string, password []byte) (string, error) {
	// ── Validate input path ────────────────────────────────────────
	linfo, err := os.Lstat(inputPath)
	if err != nil {
		return "", fmt.Errorf("cannot access encrypted file")
	}
	if linfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("symlinks are not supported for security reasons")
	}

	// ── Read encrypted file ────────────────────────────────────────
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("cannot read encrypted file")
	}

	// Minimum valid size: header + salt + nonce + GCM tag (16 bytes).
	minSize := HeaderSize + SaltSize + NonceSize + 16
	if len(data) < minSize {
		return "", errors.New("file is too small to be a valid encrypted file")
	}

	// ── Verify magic header ────────────────────────────────────────
	// SECURITY: constant-time comparison prevents timing side-channels
	// that could reveal whether a file is encrypted.
	if subtle.ConstantTimeCompare(data[:HeaderSize], []byte(MagicHeader)) != 1 {
		return "", errors.New("not a valid PSE encrypted file")
	}

	// ── Extract components from the file ───────────────────────────
	salt := data[HeaderSize : HeaderSize+SaltSize]
	nonce := data[HeaderSize+SaltSize : HeaderSize+SaltSize+NonceSize]
	ciphertext := data[HeaderSize+SaltSize+NonceSize:]

	// ── Derive key ─────────────────────────────────────────────────
	key := e.deriveKey(password, salt)
	defer ClearBytes(key)

	// ── Create AES-256-GCM cipher ──────────────────────────────────
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("decryption failed")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("decryption failed")
	}

	// ── Decrypt and verify authentication tag ──────────────────────
	// SECURITY: If the password is wrong, the Argon2id-derived key
	// will differ, and the GCM authentication tag will NOT match.
	// gcm.Open returns an error — we never produce garbled output.
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Deliberately vague error — do not reveal whether the password
		// was wrong or the ciphertext was tampered with.
		return "", errors.New("decryption failed: wrong password or corrupted file")
	}

	// ── Determine output path ──────────────────────────────────────
	ext := e.settings.Extension
	if ext == "" {
		ext = ".pse"
	}

	outputPath := inputPath
	if strings.HasSuffix(inputPath, ext) {
		// Strip the encrypted extension to restore the original name.
		outputPath = strings.TrimSuffix(inputPath, ext)
	} else {
		outputPath = inputPath + ".decrypted"
	}

	if e.settings.OutputFolder != "" {
		outputPath = filepath.Join(e.settings.OutputFolder, filepath.Base(outputPath))
	}

	// ── Ensure we don't silently overwrite an existing file ────────
	outputPath = safePath(outputPath)

	// ── Write decrypted file ───────────────────────────────────────
	if err := os.WriteFile(outputPath, plaintext, 0600); err != nil {
		return "", fmt.Errorf("failed to write decrypted file")
	}

	// Zero plaintext from memory.
	ClearBytes(plaintext)

	return outputPath, nil
}

// IsEncryptedFile checks whether a file begins with the PSE magic header.
// It reads only the first 5 bytes, so it is safe to call on very large files.
func IsEncryptedFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, HeaderSize)
	n, err := f.Read(header)
	if err != nil || n < HeaderSize {
		return false
	}

	return subtle.ConstantTimeCompare(header, []byte(MagicHeader)) == 1
}

// ClearBytes overwrites a byte slice with zeros to remove sensitive data
// (passwords, keys, plaintext) from memory.
//
// SECURITY NOTE: This is a defense-in-depth measure. Go's garbage collector
// may have already copied the data to a different memory location. For
// maximum security, consider using a mlock'd buffer, but that is outside
// the scope of this application.
func ClearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	// Prevent the compiler from optimizing away the zeroing loop.
	runtime.KeepAlive(b)
}

// safePath returns a unique file path that won't overwrite existing files.
// If the path is available, it is returned as-is. Otherwise a numeric
// counter is inserted before the final extension:
//
//	"photo.jpg.pse" → "photo.jpg_1.pse"
//	"document.pdf"  → "document_1.pdf"
func safePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 10000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}
