package testsupport

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// BashValidate runs `bash -n` on the provided content.
func BashValidate(content string) error {
	tmp, err := os.CreateTemp("", "envseed-render-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	cmd := exec.Command("bash", "-n", tmp.Name())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bash -n failed: %w", err)
	}
	return nil
}

// ParseBashDeclareLine parses a single `declare -p` line and returns the
// variable name and decoded value. Test-only helper.
func ParseBashDeclareLine(line string) (string, string, error) {
	switch {
	case strings.HasPrefix(line, "declare -- "):
		line = strings.TrimPrefix(line, "declare -- ")
	case strings.HasPrefix(line, "declare -x "):
		line = strings.TrimPrefix(line, "declare -x ")
	default:
		return "", "", fmt.Errorf("unexpected declare prefix: %q", line)
	}
	eq := strings.IndexByte(line, '=')
	if eq <= 0 {
		return "", "", fmt.Errorf("invalid declare line: %q", line)
	}
	name := line[:eq]
	raw := line[eq+1:]
	value, err := DecodeBashDeclareValue(raw)
	if err != nil {
		return "", "", fmt.Errorf("decode %s from %q: %w", name, raw, err)
	}
	return name, value, nil
}

// DecodeBashDeclareValue decodes a bash token as printed by `declare -p`.
// Supports $'...' and "..." quoting styles.
func DecodeBashDeclareValue(raw string) (string, error) {
	if strings.HasPrefix(raw, "$'") && strings.HasSuffix(raw, "'") {
		return DecodeBashDollarSingleQuoted(raw[2 : len(raw)-1])
	}
	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		// Try Go unquote first; if it fails, perform a tolerant replacement.
		if v, err := strconv.Unquote(raw); err == nil {
			return v, nil
		}
		body := raw[1 : len(raw)-1]
		body = strings.ReplaceAll(body, `\"`, `"`)
		body = strings.ReplaceAll(body, `\\`, `\`)
		body = strings.ReplaceAll(body, `\$`, `$`)
		return body, nil
	}
	return raw, nil
}

// DecodeBashDollarSingleQuoted decodes $'...' bodies using bash escape rules.
func DecodeBashDollarSingleQuoted(body string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		ch := body[i]
		if ch != '\\' {
			b.WriteByte(ch)
			continue
		}
		i++
		if i >= len(body) {
			return "", fmt.Errorf("unterminated escape")
		}
		switch body[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '0':
			b.WriteByte(0)
		case '\\':
			b.WriteByte('\\')
		case '\'':
			b.WriteByte('\'')
		case '"':
			b.WriteByte('"')
		case 'a':
			b.WriteByte('\a')
		case 'b':
			b.WriteByte('\b')
		case 'v':
			b.WriteByte('\v')
		case 'f':
			b.WriteByte('\f')
		case 'e':
			b.WriteByte(0x1b)
		case 'c':
			return "", fmt.Errorf("\\c escape not supported in decode")
		case 'x':
			if i+2 >= len(body) {
				return "", fmt.Errorf("short hex escape")
			}
			hex := body[i+1 : i+3]
			v, err := strconv.ParseUint(hex, 16, 8)
			if err != nil {
				return "", fmt.Errorf("invalid hex escape %q", hex)
			}
			b.WriteByte(byte(v))
			i += 2
		default:
			if body[i] >= '0' && body[i] <= '7' {
				j := i
				for ; j < len(body) && j < i+3 && body[j] >= '0' && body[j] <= '7'; j++ {
				}
				v, err := strconv.ParseUint(body[i:j], 8, 8)
				if err != nil {
					return "", fmt.Errorf("invalid octal escape %q", body[i:j])
				}
				b.WriteByte(byte(v))
				i = j - 1
			} else {
				b.WriteByte(body[i])
			}
		}
	}
	return b.String(), nil
}
