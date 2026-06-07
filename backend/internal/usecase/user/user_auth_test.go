package user

import (
	"testing"
)

func TestUserUseCase_Login(t *testing.T) {
	t.Parallel()

	for _, tt := range loginTestCases() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runLoginTest(t, tt)
		})
	}
}
