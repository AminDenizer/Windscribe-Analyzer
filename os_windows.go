//go:build windows

package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows/registry"
)

const simpleCryptKey uint64 = 0x4572A4ACF31A31BA

func defaultWindscribeCLIPath() string {
	candidates := []string{
		`C:\Program Files\Windscribe\windscribe-cli.exe`,
		`C:\Program Files (x86)\Windscribe\windscribe-cli.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("windscribe-cli.exe"); err == nil {
		return p
	}
	return `C:\Program Files\Windscribe\windscribe-cli.exe`
}

func prepareCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func openFileInOS(filePath string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", filePath)
	prepareCmd(cmd)
	return cmd.Start()
}

func killPlatformVPNHelpers() {
	for _, exe := range []string{"windscribewstunnel.exe", "windscribeopenvpn.exe", "windscribewireguard.exe"} {
		cmd := exec.Command("taskkill", "/F", "/IM", exe, "/T")
		prepareCmd(cmd)
		_ = cmd.Run()
	}
}

func restartPlatformWindscribeApp() {
	killPlatformVPNHelpers()
	killCmd := exec.Command("taskkill", "/F", "/IM", "Windscribe.exe")
	prepareCmd(killCmd)
	_ = killCmd.Run()
	time.Sleep(1 * time.Second)

	wsExe := `C:\Program Files\Windscribe\Windscribe.exe`
	if _, err := os.Stat(wsExe); err != nil {
		wsExe = `C:\Program Files (x86)\Windscribe\Windscribe.exe`
	}
	startCmd := exec.Command(wsExe)
	prepareCmd(startCmd)
	_ = startCmd.Start()
	time.Sleep(2500 * time.Millisecond)
}

func getKeyParts() [8]byte {
	var parts [8]byte
	for i := 0; i < 8; i++ {
		parts[i] = byte((simpleCryptKey >> (8 * i)) & 0xFF)
	}
	return parts
}

func decryptSimpleCrypt(cipherB64 string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}
	if len(raw) < 3 || raw[0] != 0x03 {
		return nil, fmt.Errorf("invalid simplecrypt header")
	}

	flags := raw[1]
	data := make([]byte, len(raw)-2)
	copy(data, raw[2:])

	keyParts := getKeyParts()
	var lastChar byte = 0
	for i := 0; i < len(data); i++ {
		curr := data[i]
		data[i] = curr ^ lastChar ^ keyParts[i%8]
		lastChar = curr
	}

	if len(data) < 1 {
		return nil, fmt.Errorf("payload too short")
	}
	data = data[1:] // chop random byte

	if (flags & 0x02) != 0 {
		if len(data) < 2 {
			return nil, fmt.Errorf("payload too short for checksum")
		}
		data = data[2:] // chop 2-byte checksum
	}

	if (flags & 0x01) != 0 {
		if len(data) < 4 {
			return nil, fmt.Errorf("payload too short for compressed len")
		}
		r, err := zlib.NewReader(bytes.NewReader(data[4:]))
		if err != nil {
			return nil, fmt.Errorf("zlib reader error: %w", err)
		}
		defer r.Close()
		decompressed, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("zlib read error: %w", err)
		}
		data = decompressed
	}

	return data, nil
}

func encryptSimpleCrypt(plaintext []byte) (string, error) {
	var compressed bytes.Buffer
	var uncompressedLen [4]byte
	binary.BigEndian.PutUint32(uncompressedLen[:], uint32(len(plaintext)))
	compressed.Write(uncompressedLen[:])

	zw, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err := zw.Write(plaintext); err != nil {
		return "", err
	}
	if err := zw.Close(); err != nil {
		return "", err
	}

	flags := byte(0x01) // compression only
	randomByte := byte(0x42)
	ba := make([]byte, 1+compressed.Len())
	ba[0] = randomByte
	copy(ba[1:], compressed.Bytes())

	keyParts := getKeyParts()
	var lastChar byte = 0
	for i := 0; i < len(ba); i++ {
		curr := ba[i] ^ keyParts[i%8] ^ lastChar
		ba[i] = curr
		lastChar = curr
	}

	result := make([]byte, 2+len(ba))
	result[0] = 0x03
	result[1] = flags
	copy(result[2:], ba)

	return base64.StdEncoding.EncodeToString(result), nil
}

func decodeQString(data []byte, offset int) (string, int, bool) {
	if offset+4 > len(data) {
		return "", offset, false
	}
	l := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4
	if l == 0xFFFFFFFF || l == 0 {
		return "", offset, true
	}
	if l%2 != 0 || offset+int(l) > len(data) {
		return "", offset, false
	}
	u16 := make([]uint16, l/2)
	for i := 0; i < int(l)/2; i++ {
		u16[i] = binary.BigEndian.Uint16(data[offset+i*2 : offset+i*2+2])
	}
	return string(utf16.Decode(u16)), offset + int(l), true
}

func encodeQString(s string) []byte {
	if s == "" {
		res := make([]byte, 4)
		binary.BigEndian.PutUint32(res, 0xFFFFFFFF)
		return res
	}
	u16 := utf16.Encode([]rune(s))
	byteLen := len(u16) * 2
	res := make([]byte, 4+byteLen)
	binary.BigEndian.PutUint32(res[:4], uint32(byteLen))
	for i, v := range u16 {
		binary.BigEndian.PutUint16(res[4+i*2:4+i*2+2], v)
	}
	return res
}

func applyPlatformConnectionTweaks(ctx context.Context, config TestConfig, logger func(string)) error {
	// 1. Configure Large TLS / stealth extra tls padding in windscribe_extra.conf
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		extraConfDir := filepath.Join(localAppData, "Windscribe", "Windscribe2")
		extraConfPath := filepath.Join(extraConfDir, "windscribe_extra.conf")
		_ = os.MkdirAll(extraConfDir, 0755)

		if config.LargeTLS {
			_ = os.WriteFile(extraConfPath, []byte("ws-stealth-extra-tls-padding\n"), 0644)
			logger("Windscribe config: Applied Large TLS padding to windscribe_extra.conf")
		} else {
			// Clear extra config file if disabled
			_ = os.WriteFile(extraConfPath, []byte(""), 0644)
			logger("Windscribe config: Cleared Large TLS padding from windscribe_extra.conf")
		}
	}

	// 2. Determine target values from config
	targetTweaks := int32(1) // Enabled
	targetAmnezia := strings.TrimSpace(config.AmneziaConfig)
	if !config.ProtocolTweaks || targetAmnezia == "None (Disabled)" || targetAmnezia == "Disabled" || targetAmnezia == "None" {
		targetTweaks = int32(2) // Disabled
		targetAmnezia = ""
	}

	targetRouting := int32(0) // Auto
	switch strings.ToLower(strings.TrimSpace(config.ServerRouting)) {
	case "regular":
		targetRouting = 1
	case "alternate", "alternative":
		targetRouting = 2
	default:
		targetRouting = 0
	}

	targetSni := strings.TrimSpace(config.CustomSNI)
	if targetSni == "-" {
		targetSni = ""
	}

	// Large TLS in engineSettings (isAPIAntiCensorship at splitOffset - 10)
	targetAntiCensorship := byte(0)
	if config.LargeTLS {
		targetAntiCensorship = byte(1)
	}

	// Internal DNS mapping (0=OS Default, 1=OpenDNS, 2=Cloudflare, 3=Google, 4=ControlD)
	targetDns := int32(3) // default Google
	switch strings.ToLower(strings.TrimSpace(config.InternalDNS)) {
	case "os default", "osdefault":
		targetDns = 0
	case "open dns", "opendns":
		targetDns = 1
	case "cloudflare":
		targetDns = 2
	case "google":
		targetDns = 3
	case "control d", "controld":
		targetDns = 4
	}

	// 3. Open Windows Registry
	regKey, err := registry.OpenKey(registry.CURRENT_USER, `Software\Windscribe\Windscribe2`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		logger(fmt.Sprintf("Windscribe registry key not accessible: %v", err))
		return nil
	}
	defer regKey.Close()

	val, _, err := regKey.GetStringValue("engineSettings")
	if err != nil {
		logger(fmt.Sprintf("Windscribe engineSettings not found in registry: %v", err))
		return nil
	}

	data, err := decryptSimpleCrypt(val)
	if err != nil {
		logger(fmt.Sprintf("Failed to decrypt Windscribe engineSettings: %v", err))
		return nil
	}

	// Find the tail offset in decrypted data
	var splitOffset = -1
	var currentAmnezia string
	var currentRouting int32
	var currentTweaks int32
	var currentEgress int32 = 1
	var currentSni string

	for offset := len(data) - 20; offset >= 0 && offset > len(data)-500; offset-- {
		pos := offset
		amnezia, nextPos, ok := decodeQString(data, pos)
		if !ok {
			continue
		}
		pos = nextPos

		if pos+12 > len(data) {
			continue
		}
		routing := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		tweaks := int32(binary.BigEndian.Uint32(data[pos+4 : pos+8]))
		egress := int32(binary.BigEndian.Uint32(data[pos+8 : pos+12]))
		if routing < 0 || routing > 2 || tweaks < 0 || tweaks > 2 || egress < 0 || egress > 2 {
			continue
		}
		pos += 12

		sni, nextPos, ok := decodeQString(data, pos)
		if !ok || nextPos != len(data) {
			continue
		}

		splitOffset = offset
		currentAmnezia = amnezia
		currentRouting = routing
		currentTweaks = tweaks
		currentEgress = egress
		currentSni = sni
		break
	}

	if splitOffset == -1 {
		logger("Windscribe engineSettings structure unrecognized, skipping injection.")
		return nil
	}

	// Current AntiCensorship / Large TLS setting in engineSettings
	currentAntiCensorship := byte(0)
	if splitOffset >= 10 {
		currentAntiCensorship = data[splitOffset-10]
	}

	// Locate current dnsPolicy in data
	dnsOffset := -1
	currentDns := int32(-1)
	for i := splitOffset - 150; i <= splitOffset-40; i++ {
		if i < 0 || i+12 > len(data) {
			continue
		}
		p := int32(binary.BigEndian.Uint32(data[i : i+4]))
		adapter := int32(binary.BigEndian.Uint32(data[i+4 : i+8]))
		ovpnLen := binary.BigEndian.Uint32(data[i+8 : i+12])
		if p >= 0 && p <= 4 && (adapter == 1 || adapter == 2) && ovpnLen == 0xFFFFFFFF {
			dnsOffset = i
			currentDns = p
			break
		}
	}

	// Check if existing settings already match
	if currentAmnezia == targetAmnezia &&
		currentRouting == targetRouting &&
		currentTweaks == targetTweaks &&
		currentSni == targetSni &&
		currentAntiCensorship == targetAntiCensorship &&
		(dnsOffset == -1 || currentDns == targetDns) {
		logger(fmt.Sprintf("Windscribe Engine Anti-Censorship settings already active (Amnezia: %q, SNI: %q, Routing: %d, Large TLS: %v, DNS: %s).", targetAmnezia, targetSni, targetRouting, config.LargeTLS, config.InternalDNS))
		return nil
	}

	// Apply in-place modifications to prefix
	if splitOffset >= 10 {
		data[splitOffset-10] = targetAntiCensorship
	}
	if dnsOffset != -1 {
		binary.BigEndian.PutUint32(data[dnsOffset:dnsOffset+4], uint32(targetDns))
	}

	// Build new data buffer
	var newData bytes.Buffer
	newData.Write(data[:splitOffset])
	newData.Write(encodeQString(targetAmnezia))

	var intBuf [4]byte
	binary.BigEndian.PutUint32(intBuf[:], uint32(targetRouting))
	newData.Write(intBuf[:])

	binary.BigEndian.PutUint32(intBuf[:], uint32(targetTweaks))
	newData.Write(intBuf[:])

	binary.BigEndian.PutUint32(intBuf[:], uint32(currentEgress))
	newData.Write(intBuf[:])

	newData.Write(encodeQString(targetSni))

	newEncrypted, err := encryptSimpleCrypt(newData.Bytes())
	if err != nil {
		logger(fmt.Sprintf("Failed to encrypt updated Windscribe settings: %v", err))
		return nil
	}

	if err := regKey.SetStringValue("engineSettings", newEncrypted); err != nil {
		logger(fmt.Sprintf("Failed to write updated Windscribe engineSettings to registry: %v", err))
		return nil
	}

	logger(fmt.Sprintf("Injected live settings into Registry: Amnezia [%s], SNI [%s], Routing [%d], Large TLS [%v], DNS [%s].", targetAmnezia, targetSni, targetRouting, config.LargeTLS, config.InternalDNS))

	// If Windscribe.exe is currently running, restart it so the Qt engine reads the new parameters
	checkCmd := exec.Command("tasklist", "/FI", "IMAGENAME eq Windscribe.exe")
	prepareCmd(checkCmd)
	out, err := checkCmd.Output()
	if err == nil && strings.Contains(string(out), "Windscribe.exe") {
		logger("Restarting Windscribe desktop app to reload Anti-Censorship engine parameters...")

		// Disconnect if currently active
		worker := NewWindscribeWorker()
		_ = worker.EnsureDisconnected(ctx)

		// Terminate Windscribe.exe
		killCmd := exec.Command("taskkill", "/F", "/IM", "Windscribe.exe")
		prepareCmd(killCmd)
		_ = killCmd.Run()
		time.Sleep(1200 * time.Millisecond)

		// Relaunch Windscribe.exe
		wsExe := `C:\Program Files\Windscribe\Windscribe.exe`
		if _, err := os.Stat(wsExe); err != nil {
			wsExe = `C:\Program Files (x86)\Windscribe\Windscribe.exe`
		}
		startCmd := exec.Command(wsExe)
		prepareCmd(startCmd)
		if err := startCmd.Start(); err == nil {
			logger("Waiting for Windscribe Engine IPC service to initialize...")
			time.Sleep(2500 * time.Millisecond)

			// Verify CLI responds
			for i := 0; i < 5; i++ {
				if _, err := worker.RunCLI(ctx, "status"); err == nil {
					break
				}
				time.Sleep(1 * time.Second)
			}
			logger("Windscribe Engine reloaded and ready with live Anti-Censorship settings.")
		}
	}

	return nil
}
