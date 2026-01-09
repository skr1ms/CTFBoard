package e2e

import (
	"context"

	"github.com/gavv/httpexpect/v2"
	"github.com/google/uuid"
)

func registerUser(e *httpexpect.Expect, username string) (email, password string) {
	email = username + "@example.com"
	password = "password123"

	e.POST("/api/v1/auth/register").
		WithJSON(map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		}).
		Expect().
		Status(201)

	return email, password
}

func login(e *httpexpect.Expect, email, password string) string {
	resp := e.POST("/api/v1/auth/login").
		WithJSON(map[string]string{
			"email":    email,
			"password": password,
		}).
		Expect().
		Status(200).
		JSON().
		Object()

	return "Bearer " + resp.Value("access_token").String().Raw()
}

func joinTeam(e *httpexpect.Expect, authToken, inviteToken string) {
	e.POST("/api/v1/teams/join").
		WithHeader("Authorization", authToken).
		WithJSON(map[string]string{
			"invite_token": inviteToken,
		}).
		Expect().
		Status(200)
}

func createChallenge(e *httpexpect.Expect, authToken string, challenge map[string]interface{}) string {
	resp := e.POST("/api/v1/admin/challenges").
		WithHeader("Authorization", authToken).
		WithJSON(challenge).
		Expect().
		Status(201).
		JSON().
		Object()

	challengeId := resp.Value("id").String().Raw()
	if challengeId == "" {
		panic("challengeId is empty after creation")
	}
	return challengeId
}

func registerAdmin(e *httpexpect.Expect, username string) (email, password, token string) {
	email, password = registerUser(e, username)
	token = login(e, email, password)

	meResp := e.GET("/api/v1/auth/me").
		WithHeader("Authorization", token).
		Expect().
		Status(200).
		JSON().
		Object()

	userId := meResp.Value("id").String().Raw()

	if err := MakeUserAdmin(userId); err != nil {
		panic(err)
	}

	token = login(e, email, password)
	return email, password, token
}

func submitFlag(e *httpexpect.Expect, authToken, challengeId, flag string) {
	e.POST("/api/v1/challenges/{id}/submit", challengeId).
		WithHeader("Authorization", authToken).
		WithJSON(map[string]string{
			"flag": flag,
		}).
		Expect().
		Status(200)
}

func MakeUserAdmin(userId string) error {
	_, err := TestDB.ExecContext(context.Background(), "UPDATE users SET role = 'admin' WHERE id = ?", uuid.MustParse(userId))
	return err
}
