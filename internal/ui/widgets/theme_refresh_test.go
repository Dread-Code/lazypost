package ui

import (
	"net/http"
	"testing"

	"lazypost/internal/httpclient"
	"lazypost/internal/ui/themes"
)

func TestWidgetsRefreshCopiedThemeStyles(t *testing.T) {
	forceTrueColor(t)
	t.Cleanup(func() { themes.DefaultTheme.Apply() })
	themes.DefaultTheme.Apply()

	palette := NewPalette(40, 10)
	namer := NewNamer()
	auth := NewAuthEditor()
	response := NewResponse(40, 10)
	sidebar := NewSidebar(nil, ".", 30, 10)
	history := NewHistory(40, 10)

	res := &httpclient.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Headers:    http.Header{},
		Body:       []byte(`{"key":"value"}`),
	}
	response.SetResponse(res)
	beforeResponse := response.body.View()

	solarized := themes.ThemeByName("solarized")
	solarized.Apply()
	palette.RefreshTheme()
	namer.RefreshTheme()
	auth.RefreshTheme()
	response.RefreshTheme()
	sidebar.RefreshTheme()
	history.RefreshTheme()

	if got := palette.list.FilterInput.TextStyle.GetForeground(); got != solarized.Input {
		t.Errorf("palette input foreground = %v, want %v", got, solarized.Input)
	}
	if got := namer.input.TextStyle.GetForeground(); got != solarized.Input {
		t.Errorf("namer input foreground = %v, want %v", got, solarized.Input)
	}
	if got := auth.token.TextStyle.GetForeground(); got != solarized.Input {
		t.Errorf("auth input foreground = %v, want %v", got, solarized.Input)
	}
	if got := response.spinner.Style.GetForeground(); got != solarized.Primary {
		t.Errorf("spinner foreground = %v, want %v", got, solarized.Primary)
	}
	if got := sidebar.list.Styles.NoItems.GetForeground(); got != solarized.Muted {
		t.Errorf("sidebar empty-state foreground = %v, want %v", got, solarized.Muted)
	}
	if got := history.list.Styles.NoItems.GetForeground(); got != solarized.Muted {
		t.Errorf("history empty-state foreground = %v, want %v", got, solarized.Muted)
	}
	if after := response.body.View(); after == beforeResponse {
		t.Error("response body did not refresh after theme change")
	}
}
