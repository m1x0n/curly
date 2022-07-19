package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"os"
	v8 "rogchap.com/v8go"
	"strings"
	"text/template"
)

func main() {
	fmt.Println("Curly")

	json2Go, err := readJsonToGo()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//fmt.Println(json2Go)

	// Grab js functions
	curl2Go, err := readCurl2Go()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//fmt.Println(curl2Go)

	// Grab curl from stdin
	curlString, err := readCurlStdIn()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Execute curl2Go with curlString on v8 engine
	goString, err := executeOnV8(json2Go, curl2Go, curlString)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//fmt.Println(goString)

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

//TODO: Embed js file as a part of binary
func readCurl2Go() (string, error) {
	return readFile("./js/curl-to-go.js")
}

//TODO: Embed js file as a part of binary
func readJsonToGo() (string, error) {
	return readFile("./js/json-to-go.js")
}

func executeOnV8(json2Go string, curl2Go string, curl string) (string, error) {
	ctx := v8.NewContext()

	// Load scripts to the main context
	ctx.RunScript(json2Go, "main.js")
	ctx.RunScript(curl2Go, "main.js")

	// Compose js function call
	script := "const result = curlToGo(`" + curl + "`)"

	// Load js function call to the main context
	ctx.RunScript(script, "main.js")

	// Evaluate js result
	val, err := ctx.RunScript("result", "value.js")

	if err != nil {
		return "", nil
	}

	return val.String(), nil
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
		"code": code,
	}

	err = tpl.Execute(&result, data)

	if err != nil {
		return "", err
	}

	return result.String(), nil
}
