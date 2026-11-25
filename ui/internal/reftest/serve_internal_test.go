package reftest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInjectPutsTheFontsFirstAndKeepsTheMode(t *testing.T) {
	for _, c := range []struct {
		page string
		xml  bool
		want string
	}{
		{"<!DOCTYPE html>\n<title>t</title>", false, "<!DOCTYPE html><style>X</style>\n<title>t</title>"},
		{"<p>quirks</p>", false, "<style>X</style><p>quirks</p>"},
		{`<?xml version="1.0"?><html xmlns="x"><head><title/></head></html>`, true,
			`<?xml version="1.0"?><html xmlns="x"><head><style>X</style><title/></head></html>`},
		{`<html xmlns="x"><body/></html>`, true, `<html xmlns="x"><style>X</style><body/></html>`},
	} {
		if got := string(Inject([]byte(c.page), c.xml, "X")); got != c.want {
			t.Errorf("Inject(%q) = %q, want %q", c.page, got, c.want)
		}
	}
}

func TestGenericsNameTheFacesEmberDrawsThemWith(t *testing.T) {
	for in, want := range map[string]string{
		"p { font-family: serif; color: red }":       `p { font-family: "Go"; color: red }`,
		"p{font: bold 20px/1 Ahem, monospace}":       `p{font: bold 20px/1 Ahem, "Go Mono"}`,
		`<div style="font-family: Ahem">`:            `<div style="font-family: Ahem">`,
		"a { font-family: cursive, sans-serif; }":    `a { font-family: "Go", "Go"; }`,
		"b { font-size: 12px; font-style: italic; }": "b { font-size: 12px; font-style: italic; }",
	} {
		if got := Generics(in); got != want {
			t.Errorf("Generics(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHandlerServesXHTMLAsXMLWithTheFonts(t *testing.T) {
	srv := httptest.NewServer(Handler(fixture))
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + "/css/css-demo/empty.xht")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Header.Get("Content-Type") != "application/xhtml+xml" ||
		!strings.Contains(string(body), "<head><style>@font-face { font-family: Ahem; src: url(/fonts/Ahem.ttf)") {
		t.Errorf("%s:\n%s", resp.Header.Get("Content-Type"), body)
	}
	resp, err = srv.Client().Get(srv.URL + "/../MANIFEST.json")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("the checkout's root file: %d", resp.StatusCode)
	}
	resp, err = srv.Client().Get(srv.URL + "/absent.html")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("an absent file: %d", resp.StatusCode)
	}
}
