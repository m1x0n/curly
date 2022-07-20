package main

import (
	"bufio"
	"bytes"
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"github.com/dop251/goja"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"os"
	v8 "rogchap.com/v8go"
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

func main() {
	// Init flag params
	// TODO: Use fancier urfave/cli
	opts := Options{}

	flag.IntVar(&opts.NumReqs, "r", 1, "Number of requests")
	flag.IntVar(&opts.ConcurrentReqs, "c", 1, "Number of concurrent requests")
	flag.DurationVar(&opts.SleepDuration, "s", 0, "Sleep time duration")
	flag.BoolVar(&opts.IsDump, "d", false, "Dump request without actual run")
	flag.Parse()

	// Grab curl from stdin
	curlString, err := readCurlStdIn()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Read aux js functions
	scripts, err := readScripts()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Execute curl2Go with curlString on v8 engine
	//goString, err := executeOnV8(curlString, scripts...)
	goString, err := executeOnGoja(curlString, scripts...)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//fmt.Println(goString)

	if len(goString) == 0 || goString == "undefined" {
		fmt.Println("Failed to convert curl properly")
		os.Exit(1)
	}

	// Make code ready to execute standalone
	goCode, err := normalizeGoCode(goString, opts)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if opts.IsDump {
		fmt.Println(goCode)
		return
	}

	// Execute(interpret) generated go code in go via
	// https://github.com/traefik/yaegi
	err = executeOnYaegi(goCode)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func readCurlStdIn() (string, error) {
	data := strings.Builder{}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		data.WriteString(scanner.Text())
		data.WriteString("\n")
	}

	return data.String(), nil
}

func readFile(name string) (string, error) {
	file, err := os.Open(name)

	if err != nil {
		return "", err
	}

	defer file.Close()

	builder := strings.Builder{}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		builder.WriteString(scanner.Text())
		builder.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return builder.String(), nil
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

func prepareJsCall(curl string) (string, error) {
	script := "const result = curlToGo(`" + curl + "`)"
	return script, nil
}

// Also works with polyfill for URLSearchParams
func executeOnV8(curl string, scripts ...string) (string, error) {
	ctx := v8.NewContext()

	defer ctx.Close()

	// Load scripts to the main context
	for _, script := range scripts {
		ctx.RunScript(script, "main.js")
	}

	// Compose js function call
	script, err := prepareJsCall(curl)

	if err != nil {
		return "", err
	}

	// Load js function call to the main context
	ctx.RunScript(script, "main.js")

	// Evaluate js result
	val, err := ctx.RunScript("result", "value.js")

	if err != nil {
		return "", err
	}

	return val.String(), nil
}

//  Turns out it requires some not implemented feature by this engine. Silent error in v8go
//  ReferenceError: URLSearchParams is not defined at renderComplex (<eval>:252:24(130))
//  So we need polyfills for this class.
//  It's working with polyfill for URLSearchParams!
func executeOnGoja(curl string, scripts ...string) (string, error) {
	vm := goja.New()

	// Load scripts to the main context
	for _, script := range scripts {
		vm.RunString(script)
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

func normalizeGoCode(code string, opts Options) (string, error) {
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

func getImports(code string) []string {
	imports := []string{
		"net/http",
		"io/ioutil",
		"fmt",
		"sync",
		"time",
	}

	if strings.Contains(code, "application/json") {
		imports = append(imports, []string{"encoding/json", "bytes"}...)
	}

	if strings.Contains(code, "url.Values{}") {
		imports = append(imports, "net/url")
	}

	if strings.Contains(code, "strings.NewReader") {
		imports = append(imports, "strings")
	}

	if strings.Contains(code, "os.Open") {
		imports = append(imports, "os")
	}

	if strings.Contains(code, "io.MultiReader") {
		imports = append(imports, "io")
	}

	if strings.Contains(code, "tls.Config") {
		imports = append(imports, "crypto/tls")
	}

	sort.Strings(imports)

	return imports
}
