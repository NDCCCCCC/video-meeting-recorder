// Cross-end SM4 verifier — decrypts a frontend-produced ciphertext with
// the backend tjfoc/gmsm/sm4 library to confirm the post-85c37c6
// contract is consistent end-to-end. If plaintext == "admin123", the
// front- and back-ends are byte-compatible for SM4-ECB.
//
// Usage: go run scripts/verify_sm4_e2e/main.go
package main

import (
	"fmt"
	"log"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
)

func main() {
	// Produced by the matching frontend script
	// (frontend/sm4-ciphertext-check.mjs) for the same key+plaintext pair.
	ciphertext := "SM4:ixuz+Lj5lE4IoutdE9Ja3g=="
	key := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
	expectedPlaintext := "admin123"

	plaintext, err := utils.DecryptPasswordECB(ciphertext, key)
	if err != nil {
		log.Fatalf("DecryptPasswordECB failed: %v", err)
	}

	fmt.Printf("ciphertext:    %s\n", ciphertext)
	fmt.Printf("key (hex):     %s\n", key)
	fmt.Printf("decrypted:     %s\n", plaintext)
	fmt.Printf("expected:      %s\n", expectedPlaintext)

	if plaintext != expectedPlaintext {
		log.Fatalf("MISMATCH: decrypted %q != expected %q", plaintext, expectedPlaintext)
	}
	fmt.Println("\nOK: front- and back-end SM4 ciphertexts decrypt to the same plaintext")
}
