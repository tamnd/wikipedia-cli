package dump

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Sums maps the name of a dump file to its sha1 digest in hex.
type Sums map[string]string

// ParseSums reads a sha1sums file of a dump run. Each line has a digest, two
// spaces and a file name.
func ParseSums(r io.Reader) (Sums, error) {
	out := Sums{}
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 2 || len(f[0]) != 40 {
			return nil, fmt.Errorf("dump: sums line %d is not a digest and a name", n)
		}
		if _, err := hex.DecodeString(f[0]); err != nil {
			return nil, fmt.Errorf("dump: sums line %d: %v", n, err)
		}
		out[strings.TrimPrefix(f[1], "*")] = strings.ToLower(f[0])
	}
	return out, sc.Err()
}

// FileSha1 returns the sha1 digest of a file in hex.
func FileSha1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Check compares a file with its digest in s. The file is found by its base
// name. It returns an error when s has no digest for the file or when the
// digests are not equal.
func (s Sums) Check(path string) error {
	name := filepath.Base(path)
	want, ok := s[name]
	if !ok {
		return fmt.Errorf("dump: no sha1 for %s", name)
	}
	got, err := FileSha1(path)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("dump: sha1 of %s is %s, want %s", name, got, want)
	}
	return nil
}
