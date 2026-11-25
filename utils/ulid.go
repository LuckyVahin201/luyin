package utils

import (
    "github.com/google/uuid"
    "github.com/oklog/ulid/v2"
)

func ToUUID(id ulid.ULID) uuid.UUID {
    return uuid.UUID(id)
}

func ToULID(id uuid.UUID) ulid.ULID {
    return ulid.ULID(id)
}

type ulidScanner struct {
	dest *ulid.ULID
}

func ScanULID(dest *ulid.ULID) *ulidScanner {
	return &ulidScanner{dest: dest}
}

