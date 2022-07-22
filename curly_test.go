package main

import (
	"github.com/m1x0n/curly/pkg"
	"os"
	"strings"
	"testing"
)

// Via the article on how to mock os.Stdin and os.Stdout
// https://eli.thegreenplace.net/2020/faking-stdin-and-stdout-in-go/
// This might be done via faking app.Writer and app.Reader
func TestSimpleCurlConversion(t *testing.T) {
	curl := `curl -X GET https://example.com`

	fakeIO, _ := fakestdio.New(curl)
	fakeIO.CloseStdin()

	app := createApp()

	os.Args = []string{"curly", "-d"}

	err := app.Run(os.Args)

	if err != nil {
		t.Fatalf("app Run error: %s", err)
	}

	result, err := fakeIO.ReadAndRestore()

	if err != nil {
		t.Fatalf("Output read error: %s", err)
	}

	resultString := string(result)

	if !strings.Contains(resultString, `http.Get("https://example.com")`) {
		t.Fatalf("cURL to go conversion failed. Resuled in %s", resultString)
	}
}

func TestComplexCurlConversion(t *testing.T) {
	curl := `curl -H "Content-Type: application/json" -H "Authorization: Bearer b7d03a6947b217efb6f3ec3bd3504582" -d '{"type":"A","name":"www","data":"162.10.66.0","priority":null,"port":null,"weight":null}' "https://api.digitalocean.com/v2/domains/example.com/records"`

	fakeIO, _ := fakestdio.New(curl)
	fakeIO.CloseStdin()

	app := createApp()

	os.Args = []string{"curly", "-d"}

	err := app.Run(os.Args)

	if err != nil {
		t.Fatalf("app Run error: %s", err)
	}

	result, err := fakeIO.ReadAndRestore()

	if err != nil {
		t.Fatalf("Output read error: %s", err)
	}

	resultString := string(result)

	if !strings.Contains(resultString, `http.NewRequest("POST", "https://api.digitalocean.com/v2/domains/example.com/records", body)`) {
		t.Fatalf("cURL to go conversion failed. No request created. Resuled in %s", resultString)
	}

	if !strings.Contains(resultString, `http.DefaultClient.Do`) {
		t.Fatalf("cURL to go conversion failed. No http client call found. Resuled in %s", resultString)
	}

	if !strings.Contains(resultString, `req.Header.Set("Content-Type", "application/json")`) {
		t.Fatalf("cURL to go conversion failed. No Content-Type header found. Resuled in %s", resultString)
	}

	if !strings.Contains(resultString, `req.Header.Set("Authorization", "Bearer b7d03a6947b217efb6f3ec3bd3504582")`) {
		t.Fatalf("cURL to go conversion failed. No Authorization header found. Resuled in %s", resultString)
	}
}
