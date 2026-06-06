package repo

import (
	"context"
	"fmt"
	"hash/fnv"
	"slices"
)

type AdvisoryLocker interface {
	AcquireAdvisoryLock(ctx context.Context, lockKey int64) error
}

type RegistrationLockScope string

const (
	RegistrationLockEmail    RegistrationLockScope = "email"
	RegistrationLockUsername RegistrationLockScope = "username"
)

type RegistrationAdvisoryLock struct {
	Label string
	Scope RegistrationLockScope
	Value string
}

type keyedRegistrationAdvisoryLock struct {
	label string
	key   int64
}

const maxPositiveAdvisoryLockKey = 1<<63 - 1

// AcquireRegistrationAdvisoryLocks acquires transaction-scoped advisory locks for
// user registration uniqueness checks in deterministic key order.
func AcquireRegistrationAdvisoryLocks(ctx context.Context, locker AdvisoryLocker, locks ...RegistrationAdvisoryLock) error {
	keyedLocks := make([]keyedRegistrationAdvisoryLock, 0, len(locks))

	for _, lock := range locks {
		key, err := registrationAdvisoryKey(lock.Scope, lock.Value)
		if err != nil {
			return err
		}

		label := lock.Label
		if label == "" {
			label = string(lock.Scope)
		}

		keyedLocks = append(keyedLocks, keyedRegistrationAdvisoryLock{label: label, key: key})
	}

	slices.SortFunc(keyedLocks, func(a, b keyedRegistrationAdvisoryLock) int {
		if a.key < b.key {
			return -1
		}

		if a.key > b.key {
			return 1
		}

		return 0
	})

	var lastKey int64

	for i, lock := range keyedLocks {
		if i > 0 && lock.key == lastKey {
			continue
		}

		if err := locker.AcquireAdvisoryLock(ctx, lock.key); err != nil {
			return fmt.Errorf("AcquireAdvisoryLock(%s): %w", lock.label, err)
		}

		lastKey = lock.key
	}

	return nil
}

func registrationAdvisoryKey(scope RegistrationLockScope, value string) (int64, error) {
	var prefix string

	switch scope {
	case RegistrationLockEmail:
		prefix = "reg:email:"
	case RegistrationLockUsername:
		prefix = "reg:username:"
	default:
		return 0, fmt.Errorf("unknown registration advisory lock scope %q", scope)
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(prefix))
	_, _ = h.Write([]byte(value))

	return int64(min(h.Sum64(), uint64(maxPositiveAdvisoryLockKey))), nil
}
