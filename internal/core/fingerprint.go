package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Fingerprint hashes configured inputs together with the configuration and contract.
func Fingerprint(root string, config *Config, contractPath string) (string, error) {
	paths, err := relevantPaths(root, config.Inputs)
	if err != nil {
		return "", err
	}
	if err := addFingerprintPath(paths, root, filepath.Join(root, "pbt.config.json")); err != nil {
		return "", err
	}
	if err := addFingerprintPath(paths, root, contractPath); err != nil {
		return "", err
	}

	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)

	hash := sha256.New()
	for _, name := range names {
		if _, err := hash.Write([]byte(name)); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", err
		}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if os.IsNotExist(err) {
			if _, err := hash.Write([]byte("missing\x00")); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte("file\x00")); err != nil {
			return "", err
		}
		if _, err := hash.Write(content); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func relevantPaths(root string, inputs []string) (map[string]struct{}, error) {
	paths := map[string]struct{}{}
	if err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if excludedLocalPath(relative) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		matched, err := matchesInputs(relative, inputs)
		if err != nil {
			return err
		}
		if matched {
			paths[relative] = struct{}{}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	tracked, err := runGit("-C", root, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	for _, name := range strings.Split(tracked, "\x00") {
		if name == "" || excludedLocalPath(name) {
			continue
		}
		matched, err := matchesInputs(filepath.ToSlash(name), inputs)
		if err != nil {
			return nil, err
		}
		if matched {
			paths[filepath.ToSlash(name)] = struct{}{}
		}
	}
	return paths, nil
}

func addFingerprintPath(paths map[string]struct{}, root, filePath string) error {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, filePath)
	}
	relative, err := filepath.Rel(root, filePath)
	if err != nil {
		return err
	}
	relative = filepath.ToSlash(relative)
	if relative == ".." || strings.HasPrefix(relative, "../") {
		return fmt.Errorf("fingerprint path %q is outside project root", filePath)
	}
	paths[relative] = struct{}{}
	return nil
}

func excludedLocalPath(filePath string) bool {
	filePath = filepath.ToSlash(filePath)
	return filePath == ".git" || strings.HasPrefix(filePath, ".git/") ||
		filePath == ".pbt/state" || strings.HasPrefix(filePath, ".pbt/state/") ||
		filePath == ".pbt/runs" || strings.HasPrefix(filePath, ".pbt/runs/")
}

func matchesInputs(name string, inputs []string) (bool, error) {
	for _, input := range inputs {
		matched, err := matchInput(strings.TrimPrefix(filepath.ToSlash(input), "./"), name)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func matchInput(pattern, name string) (bool, error) {
	if !strings.Contains(pattern, "**") {
		return path.Match(pattern, name)
	}

	var expression strings.Builder
	expression.WriteString("^")
	for index := 0; index < len(pattern); {
		switch pattern[index] {
		case '*':
			if index+1 < len(pattern) && pattern[index+1] == '*' {
				if index+2 < len(pattern) && pattern[index+2] == '/' {
					expression.WriteString("(?:.*/)?")
					index += 3
				} else {
					expression.WriteString(".*")
					index += 2
				}
				continue
			}
			expression.WriteString("[^/]*")
		case '?':
			expression.WriteString("[^/]")
		default:
			expression.WriteString(regexp.QuoteMeta(string(pattern[index])))
		}
		index++
	}
	expression.WriteString("$")
	compiled, err := regexp.Compile(expression.String())
	if err != nil {
		return false, err
	}
	return compiled.MatchString(name), nil
}
