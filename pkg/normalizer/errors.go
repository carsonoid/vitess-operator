package normalizer

import (
	"errors"
)

type ValidationError error

var (
	ValidationErrorNoLockserver ValidationError = errors.New("No Lockserver in Cluster")
	ValidationErrorNoCells      ValidationError = errors.New("No Cells in Cluster")
	ValidationErrorNoShards     ValidationError = errors.New("No Shards in Cluster")
	ValidationErrorNoTablets    ValidationError = errors.New("No Tablets in Cluster")
	ValidationErrorNoKeyspaces  ValidationError = errors.New("No Keyspaces in Cluster")
)

var ClientError = errors.New("Client Error")
