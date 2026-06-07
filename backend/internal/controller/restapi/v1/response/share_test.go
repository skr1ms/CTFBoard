package response

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestSolveShareHTML_EscapesUserControlledFields(t *testing.T) {
	html, err := SolveShareHTML(&usecase.SolveShare{
		SolveID:           uuid.New(),
		TeamName:          `<script>alert("team")</script>`,
		Username:          `<img src=x onerror=alert(1)>`,
		ChallengeTitle:    `orbit & <b>root</b>`,
		ChallengeCategory: `web<script>`,
		CTFName:           `Astro <CTF>`,
		CTFDescription:    `desc "quote" & <tag>`,
		PointsAtSolve:     100,
	})
	require.NoError(t, err)

	assert.NotContains(t, html, `<script>alert("team")</script>`)
	assert.NotContains(t, html, `<img src=x onerror=alert(1)>`)
	assert.Contains(t, html, `&lt;script&gt;alert(&#34;team&#34;)&lt;/script&gt;`)
	assert.Contains(t, html, `orbit &amp; &lt;b&gt;root&lt;/b&gt;`)
	assert.Contains(t, html, `desc &#34;quote&#34; &amp; &lt;tag&gt;`)
}
