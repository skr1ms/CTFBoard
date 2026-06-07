package response

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

const solveShareHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.TeamName}} solved {{.ChallengeTitle}} | {{.CTFName}}</title>
  <meta name="description" content="{{.Description}}">
  <meta property="og:type" content="website">
  <meta property="og:title" content="{{.TeamName}} solved {{.ChallengeTitle}}">
  <meta property="og:description" content="{{.Description}}">
  {{if .CTFLogo}}<meta property="og:image" content="{{.CTFLogo}}">{{end}}
  <meta name="twitter:card" content="{{if .CTFLogo}}summary_large_image{{else}}summary{{end}}">
  <style>
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; color: #111827; background: #f8fafc; }
    main { width: min(640px, calc(100vw - 32px)); padding: 32px; border: 1px solid #e5e7eb; border-radius: 8px; background: #fff; box-shadow: 0 10px 30px rgba(15, 23, 42, .08); }
    h1 { margin: 0 0 12px; font-size: 28px; line-height: 1.2; }
    p { margin: 0 0 16px; color: #4b5563; line-height: 1.55; }
    dl { display: grid; grid-template-columns: max-content 1fr; gap: 8px 16px; margin: 24px 0; }
    dt { color: #6b7280; }
    dd { margin: 0; font-weight: 600; }
    a { color: #2563eb; text-decoration: none; font-weight: 600; }
  </style>
</head>
<body>
  <main>
    <h1>{{.TeamName}} solved {{.ChallengeTitle}}</h1>
    <p>{{.Description}}</p>
    <dl>
      <dt>Category</dt><dd>{{.ChallengeCategory}}</dd>
      <dt>Points</dt><dd>{{.PointsAtSolve}}</dd>
      <dt>Solver</dt><dd>{{.Username}}</dd>
      <dt>CTF</dt><dd>{{.CTFName}}</dd>
    </dl>
    {{if .RegisterURL}}<p><a href="{{.RegisterURL}}">Join the competition</a></p>{{end}}
  </main>
</body>
</html>`

type solveShareView struct {
	TeamName          string
	Username          string
	ChallengeTitle    string
	ChallengeCategory string
	CTFName           string
	CTFLogo           string
	RegisterURL       string
	Description       string
	PointsAtSolve     int
}

func FromShareLink(link *usecase.ShareLink) openapi.ShareResponse {
	return openapi.ShareResponse{
		Type:    openapi.ShareResponseType(link.Type),
		URL:     link.URL,
		SolveID: link.SolveID,
	}
}

func SolveShareHTML(share *usecase.SolveShare) (string, error) {
	view := solveShareView{
		TeamName:          share.TeamName,
		Username:          share.Username,
		ChallengeTitle:    share.ChallengeTitle,
		ChallengeCategory: share.ChallengeCategory,
		CTFName:           share.CTFName,
		CTFLogo:           share.CTFLogo,
		RegisterURL:       share.RegisterURL,
		Description:       solveShareDescription(share),
		PointsAtSolve:     share.PointsAtSolve,
	}

	tmpl, err := template.New("solve-share").Parse(solveShareHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("SolveShareHTML - Parse: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, view); err != nil {
		return "", fmt.Errorf("SolveShareHTML - Execute: %w", err)
	}

	return buf.String(), nil
}

func solveShareDescription(share *usecase.SolveShare) string {
	if share.CTFDescription != "" {
		return share.CTFDescription
	}

	return fmt.Sprintf("%s solved %s for %d points.", share.TeamName, share.ChallengeTitle, share.PointsAtSolve)
}
