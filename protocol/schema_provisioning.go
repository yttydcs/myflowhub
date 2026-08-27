package protocol

import (
	"encoding/base64"
	"errors"
	"fmt"
)

const SchemaProvisioningPermitV1 = "mfh.provisioning.permit.v1"

type ProvisioningPermitV1 struct {
	Version         int    `json:"version"`
	PermitID        string `json:"permit_id"`
	ParentNodeID    string `json:"parent_node_id"`
	ChildNodeID     string `json:"child_node_id"`
	ChildPublicKey  string `json:"child_public_key"`
	Role            string `json:"role"`
	IssuedAtUnixMS  int64  `json:"issued_at_unix_ms"`
	ExpiresAtUnixMS int64  `json:"expires_at_unix_ms"`
	MaxUses         int    `json:"max_uses"`
	Signature       string `json:"signature"`
}

func (p ProvisioningPermitV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := validateHexID("permit_id", p.PermitID, 16); err != nil {
		return err
	}
	if err := validateNodeIDText("parent_node_id", p.ParentNodeID); err != nil {
		return err
	}
	if err := validateNodeIDText("child_node_id", p.ChildNodeID); err != nil {
		return err
	}
	if err := validateText("role", p.Role, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if p.IssuedAtUnixMS <= 0 || p.ExpiresAtUnixMS <= p.IssuedAtUnixMS {
		return errors.New("permit time range is invalid")
	}
	if p.MaxUses != 1 {
		return errors.New("permit max_uses must be one")
	}
	key, err := base64.RawStdEncoding.DecodeString(p.ChildPublicKey)
	if err != nil || len(key) != 32 {
		return errors.New("child_public_key must be a raw-base64 Ed25519 public key")
	}
	signature, err := base64.RawStdEncoding.DecodeString(p.Signature)
	if err != nil || len(signature) != 64 {
		return fmt.Errorf("signature must be a raw-base64 Ed25519 signature")
	}
	return nil
}
