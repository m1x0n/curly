package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/dop251/goja"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"os"
	v8 "rogchap.com/v8go"
	"sort"
	"strings"
	"text/template"
)

func main() {
	fmt.Println("Curly")

	scripts, err := readScripts()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Grab curl from stdin
	curlString, err := readCurlStdIn()

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
	goCode, err := normalizeGoCode(goString)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(goCode)

	// Execute generated go code in go via https://github.com/traefik/yaegi
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

// TODO: Embed js files
func readScripts() ([]string, error) {
	scriptPath := "./js"

	scripts := []string{
		"json-to-go.js",
		"curl-to-go.js",
		"url-search-params.js",
	}

	content := make([]string, 0)

	for _, script := range scripts {
		scriptContent, err := readFile(scriptPath + "/" + script)

		if err != nil {
			return nil, err
		}

		content = append(content, scriptContent)
	}

	return content, nil
}

func prepareJsCall(curl string) (string, error) {
	// Compose js function call

	script := "const result = curlToGo(`" + curl + "`)"
	return script, nil

	//tpl, err := template.ParseFiles("./templates/js.tmpl")
	//
	//if err != nil {
	//	return "", err
	//}
	//
	//var result bytes.Buffer
	//
	//data := map[string]interface{}{
	//	"curl": curl,
	//}
	//
	//err = tpl.Execute(&result, data)
	//
	//if err != nil {
	//	return "", err
	//}
	//
	//script := result.String()
	//
	//return script, nil
}

func executeOnV8(curl string, scripts ...string) (string, error) {
	ctx := v8.NewContext()

	defer ctx.Close()

	// Load scripts to the main context
	for _, script := range scripts {
		ctx.RunScript(script, "main.js")
	}

	// Compose js function call
	//FIXME: string like "key1=value+1&key2=value%3A2" crashes on v8
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
//  So we need polyfills for this class
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

// TODO: pass options, make it embed
func normalizeGoCode(code string) (string, error) {
	tpl, err := template.ParseFiles("./templates/request.tmpl")

	if err != nil {
		return "", err
	}

	var result bytes.Buffer

	// Data for template must be a struct/map
	// TODO: add options here and add conditions to template
	data := map[string]interface{}{
		"code":    code,
		"imports": getImports(code),
	}

	err = tpl.Execute(&result, data)

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

// Escaping of quotes is fucked up

// 1. Try exec in https://github.com/dop251/goja

// 2. Debug: -> Add intermediate function for to original js and call it
