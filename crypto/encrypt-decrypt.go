/*
Copyright 2025 Google.

This software is provided as-is, without warranty or representation for any use or purpose.
*/
package crypto

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"

	"github.com/google/tink/go/aead"
	"github.com/google/tink/go/core/registry"
	"github.com/google/tink/go/integration/gcpkms"
	log "github.com/sirupsen/logrus"
)

func Base64MD5Hash(byteStream []byte) string {
	hashProvider := md5.New()
	var base64MD5Hash string

	_, err := hashProvider.Write(byteStream)
	if err != nil {
		log.Errorf("Error computing MD5 hash:%v", err)
		return base64MD5Hash
	}
	md5Hash := hashProvider.Sum(nil) // Get the computed MD5 hash as a byte array

	// Step 2: Encode the MD5 hash as Base64
	base64MD5Hash = base64.StdEncoding.EncodeToString(md5Hash)

	// Print the result
	log.Debugf("Base64-encoded MD5 hash:%v", base64MD5Hash)
	return base64MD5Hash
}

// Encrypt bytes with KMS key referenced by resourceName in the format:
// projects/<projectname>/locations/<location>/keyRings/<project>/cryptoKeys/<key-ring>/cryptoKeyVersions/1
func EncryptBytes(ctx context.Context, resourceName string, bytesToEncrypt []byte) ([]byte, error) {
	// Construct the full key URI for Google Cloud KMS
	//projects/<projectname>/locations/<location>/keyRings/<project>/cryptoKeys/<key-ring>/cryptoKeyVersions/1
	keyURI := fmt.Sprintf("gcp-kms://%s", resourceName)

	// Create a KMS client
	kmsClient, err := gcpkms.NewClientWithOptions(ctx, keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %v", err)
	}

	// Create a KMS AEAD client
	kmsAEAD, err := kmsClient.GetAEAD(keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS AEAD client: %v", err)
	}

	// 2. Register the KMS AEAD primitive wrapper.
	registry.RegisterKMSClient(kmsClient)

	// 3. Create the KMS-backed envelope AEAD.
	envAEAD := aead.NewKMSEnvelopeAEAD2(aead.AES256GCMKeyTemplate(), kmsAEAD)
	if envAEAD == nil {
		return nil, fmt.Errorf("failed to create KMS AEAD envelope: %v", err)
	}

	// Encrypt the bytes
	aad := []byte("")
	encryptedBytes, err := envAEAD.Encrypt(bytesToEncrypt, aad)
	if err != nil {
		return nil, fmt.Errorf("error encrypting data: %v", err)
	}

	return encryptedBytes, nil
}

// Decrypts bytes with using KMS key referenced by resourceName in the format:
// projects/<projectname>/locations/<location>/keyRings/<project>/cryptoKeys/<key-ring>/cryptoKeyVersions/1
func DecryptBytes(ctx context.Context, resourceName string, bytesToDecrypt []byte) ([]byte, error) {
	// Construct the full key URI for Google Cloud KMS
	keyURI := fmt.Sprintf("gcp-kms://%s", resourceName)

	// Create a KMS client
	kmsClient, err := gcpkms.NewClientWithOptions(ctx, keyURI /*, option.WithCredentialsFile("path/to/credentials.json")*/)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS client: %v", err)
	}

	// Create a KMS AEAD client
	kmsAEAD, err := kmsClient.GetAEAD(keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS AEAD client: %v", err)
	}

	// Register the KMS AEAD primitive wrapper.
	registry.RegisterKMSClient(kmsClient)

	// Create the KMS-backed envelope AEAD.
	envAEAD := aead.NewKMSEnvelopeAEAD2(aead.AES256GCMKeyTemplate(), kmsAEAD)
	if envAEAD == nil {
		return nil, fmt.Errorf("failed to create KMS AEAD envelope: %v", err)
	}
	// Decrypt bytes with KMS key
	aad := []byte("")
	decryptedBytes, err := envAEAD.Decrypt(bytesToDecrypt, aad)
	if err != nil {
		return nil, fmt.Errorf("error encrypting data: %v", err)
	}
	return decryptedBytes, nil
}
