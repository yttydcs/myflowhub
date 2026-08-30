package protocol

import "testing"

func TestAdmissionIssuePermitDoesNotAcceptClientNodeID(t *testing.T) {
	request := AdmissionIssuePermitV1{
		Version:                    SchemaVersionV1,
		RequestID:                  "00112233445566778899aabbccddeeff",
		DevicePublicKeyFingerprint: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		TargetNodeID:               "42",
		AdmissionProfile:           "leaf",
		TTLMS:                      60_000,
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionSubmitRequiresParentAndTranscript(t *testing.T) {
	request := AdmissionSubmitV1{
		Version:          SchemaVersionV1,
		RequestID:        "00112233445566778899aabbccddeeff",
		DevicePublicKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		ParentNodeID:     "42",
		ParentPublicKey:  "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		TranscriptDigest: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	request.ParentNodeID = ""
	if err := request.Validate(); err == nil {
		t.Fatal("expected missing parent to fail")
	}
}

func TestAdmissionListCursorIsAnOpaqueEnrollmentObjectID(t *testing.T) {
	query := AdmissionListV1{Version: 1, Cursor: "00112233445566778899aabbccddeeff", Limit: 10}
	if err := query.Validate(); err != nil {
		t.Fatal(err)
	}
	query.Cursor = "not-a-cursor"
	if err := query.Validate(); err == nil {
		t.Fatal("malformed list cursor was accepted")
	}
}
