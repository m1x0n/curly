package main

import (
	"bufio"
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"github.com/dop251/goja"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/urfave/cli/v2"
	"go/format"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

type Options struct {
	NumReqs        int
	ConcurrentReqs int
	SleepDuration  time.Duration
	IsDump         bool
}

//go:embed js/json-to-go.js js/curl-to-go.js js/url-search-params.js
var js embed.FS

//go:embed templates/request.tmpl
var tpl embed.FS

// version will be overridden by build
var version = "latest"

func main() {
	app := createApp()

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createApp() *cli.App {
	opts := Options{}

	return &cli.App{
		Name:      "curly",
		Version:   version,
		Usage:     "Converts cURL command from STDIN to golang code and executes it",
		UsageText: `curly [-h|--help] [-v|--version] [-r <value>] [-c <value>] [-s <value>] [-d] <command> [<args>]`,
		Authors: []*cli.Author{
			{
				Name:  "m1x0n",
				Email: "mmorozovm@gmail.com",
			},
		},
		CustomAppHelpTemplate: getHelpTemplate(),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "r",
				Value:       1,
				Usage:       "Number of requests",
				Destination: &opts.NumReqs,
			},
			&cli.IntFlag{
				Name:        "c",
				Value:       1,
				Usage:       "Number of concurrent requests",
				Destination: &opts.ConcurrentReqs,
			},
			&cli.DurationFlag{
				Name:        "s",
				Value:       0,
				Usage:       "Sleep duration",
				Destination: &opts.SleepDuration,
			},
			&cli.BoolFlag{
				Name:        "d",
				Value:       false,
				Usage:       "Dump generated golang code",
				Destination: &opts.IsDump,
			},
		},
		Action: func(cCtx *cli.Context) error {
			return runCurly(&opts)
		},
	}
}

func runCurly(opts *Options) error {
	// Grab curl from stdin
	curlString, err := readCurl()

	if err != nil {
		return err
	}

	// Read aux js functions
	scripts, err := readScripts()

	if err != nil {
		return err
	}

	// Execute curl2Go with curlString on v8 engine
	goString, err := executeOnGoja(curlString, scripts...)

	if err != nil {
		return err
	}

	if len(goString) == 0 || goString == "undefined" {
		return fmt.Errorf("failed to convert curl properly")
	}

	// Make code ready to execute standalone
	goCode, err := normalizeGoCode(goString, opts)

	if err != nil {
		return err
	}

	// Beautify generated code
	goCode, err = beautifyGoCode(goCode)

	if err != nil {
		return err
	}

	if opts.IsDump {
		fmt.Println(goCode)
		return nil
	}

	// Execute(interpret) generated go code in go via
	err = executeOnYaegi(goCode)

	return err
}

func readCurl() (string, error) {
	data := strings.Builder{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		data.WriteString(scanner.Text())
		data.WriteString("\n")
	}

	return data.String(), nil
}

func readScripts() ([]string, error) {
	files := []string{
		"js/json-to-go.js",
		"js/curl-to-go.js",
		"js/url-search-params.js",
	}

	content := make([]string, 0)

	for _, file := range files {
		scriptContent, err := js.ReadFile(file)

		if err != nil {
			return nil, err
		}

		content = append(content, string(scriptContent))
	}

	return content, nil
}

// Turns out it requires some not implemented feature by this engine. Silent error in v8go
// ReferenceError: URLSearchParams is not defined at renderComplex (<eval>:252:24(130))
// So we need polyfills for this class.
// It's working with polyfill for URLSearchParams!
func executeOnGoja(curl string, scripts ...string) (string, error) {
	vm := goja.New()

	// Load scripts to the main context
	for _, script := range scripts {
		_, err := vm.RunString(script)
		if err != nil {
			return "", err
		}
	}

	// Get curl2GoFn callable from VM
	curlToGoFn, isOk := goja.AssertFunction(vm.Get("curlToGo"))

	if !isOk {
		return "", fmt.Errorf("goja: Failed to locate curlToGo() function in global VM context")
	}

	result, err := curlToGoFn(goja.Undefined(), vm.ToValue(curl))

	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func executeOnYaegi(code string) error {
	i := interp.New(interp.Options{})
	var err error

	err = i.Use(stdlib.Symbols)

	if err != nil {
		return err
	}

	_, err = i.Eval(code)

	return err
}

func normalizeGoCode(code string, opts *Options) (string, error) {
	t, err := template.ParseFS(tpl, "templates/request.tmpl")

	if err != nil {
		return "", err
	}

	var result bytes.Buffer

	// Data for template must be a struct/map
	data := map[string]interface{}{
		"code":    code,
		"imports": getImports(code),
		"opts":    opts,
	}

	err = t.Execute(&result, data)

	if err != nil {
		return "", err
	}

	return result.String(), nil
}

func beautifyGoCode(code string) (string, error) {
	//1. Replace '// handle err'
	beautified := strings.ReplaceAll(code, "// handle err", "fmt.Println(err)\n\treturn")

	// 2. Apply go format
	formatted, err := format.Source([]byte(beautified))

	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

func getImports(code string) []string {
	imports := []string{
		"net/http",
		"io",
		"fmt",
		"sync",
		"time",
	}

	patternMap := map[string][]string{
		"application/json":  {"encoding/json", "bytes"},
		"url.Values{}":      {"net/url"},
		"strings.NewReader": {"strings"},
		"os.Open":           {"os"},
		"io.MultiReader":    {"io"},
		"tls.Config":        {"crypto/tls"},
	}

	for pattern, importList := range patternMap {
		if strings.Contains(code, pattern) {
			imports = append(imports, importList...)
		}
	}
	sort.Strings(imports)

	return imports
}

func getHelpTemplate() string {
	examples := `
EXAMPLES:
	1. Read cURL command and run it via curly with default params:

			echo "curl -X GET https://example.com" | curly

	2. Read cURL command and dump generated go code without execution:

			echo "curl -X GET https://example.com" | curly -d

	3. Read cURL command from clipboard and run generated code in 50 requests with 5 concurrency:
		
			xclip -o | curly -r 50 -c 5

	4. Read cURL command from file and run generated code in 10 requests with 1 concurrency and 
	   sleep duration of 1 second:
		
			cat curl.txt | curly -r 10 -s 1s
`
	return fmt.Sprintf("%s %s", cli.AppHelpTemplate, examples)
}
