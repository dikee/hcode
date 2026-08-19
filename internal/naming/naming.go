// Package naming generates instance and deploy-key names.
package naming

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func tokenHex(nbytes int) string {
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// GenerateInstanceName builds "<repo>-<random>".
func GenerateInstanceName(repoName string) string {
	return fmt.Sprintf("%s-%s", repoName, tokenHex(3))
}

// GenerateKeyTitle builds "hcode-<instance>-<repo>-<random>".
func GenerateKeyTitle(instanceName, repoName string) string {
	return fmt.Sprintf("hcode-%s-%s-%s", instanceName, repoName, tokenHex(3))
}
