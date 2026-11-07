package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const docsPage = "https://seller.ggsel.com/docs/"

func main() {
	write := flag.Bool("write", false, "overwrite testdata snapshots from the live docs")
	showDiff := flag.Bool("diff", false, "print a short contract diff")
	flag.Parse()
	if err := run(*write, *showDiff); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(write, showDiff bool) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	v1Path := filepath.Join(root, "testdata", "seller-api-v1.openapi.json")
	v2Path := filepath.Join(root, "testdata", "seller-api-v2.openapi.json")
	today := time.Now().UTC().Format("2006-01-02")
	liveV1, liveV2, err := build(today)
	if err != nil {
		return err
	}
	if write {
		if err := writeDoc(v1Path, liveV1); err != nil {
			return err
		}
		if err := writeDoc(v2Path, liveV2); err != nil {
			return err
		}
		fmt.Println(filepath.Base(v1Path))
		fmt.Println(filepath.Base(v2Path))
		return nil
	}
	currentV1, err := readDoc(v1Path)
	if err != nil {
		return err
	}
	currentV2, err := readDoc(v2Path)
	if err != nil {
		return err
	}
	fmt.Printf("v1 %d operations, v2 %d operations\n", operations(liveV1), operations(liveV2))
	parts := diffParts(currentV1, liveV1, currentV2, liveV2)
	if len(parts) == 0 {
		fmt.Println("docs match the live site")
		return nil
	}
	if showDiff {
		text := strings.Join(parts, "\n")
		lines := strings.Split(text, "\n")
		if len(lines) > 200 {
			hidden := len(lines) - 200
			lines = append(lines[:200], fmt.Sprintf("... truncated, %d more lines", hidden))
		}
		fmt.Println(strings.Join(lines, "\n"))
	} else {
		fmt.Println("docs differ from the live site")
	}
	os.Exit(1)
	return nil
}

func diffParts(currentV1, liveV1, currentV2, liveV2 map[string]any) []string {
	var parts []string
	for _, item := range []struct {
		name    string
		current map[string]any
		live    map[string]any
	}{
		{"seller-api-v1", currentV1, liveV1},
		{"seller-api-v2", currentV2, liveV2},
	} {
		if bytes.Equal(canonical(contractOf(item.current)), canonical(contractOf(item.live))) {
			continue
		}
		parts = append(parts, unified(item.name, canonical(contractOf(item.current)), canonical(contractOf(item.live))))
	}
	return parts
}

func build(today string) (map[string]any, map[string]any, error) {
	html, err := get(docsPage)
	if err != nil {
		return nil, nil, err
	}
	runtimeURL, mainURL, err := bundleURLs(html)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := get(runtimeURL)
	if err != nil {
		return nil, nil, err
	}
	mainJS, err := get(mainURL)
	if err != nil {
		return nil, nil, err
	}
	base, files, err := chunkFiles(runtime)
	if err != nil {
		return nil, nil, err
	}
	items, err := routes(mainJS)
	if err != nil {
		return nil, nil, err
	}
	var shared map[string]struct{}
	for i, item := range items {
		ids := map[string]struct{}{}
		for _, id := range item.chunks {
			ids[id] = struct{}{}
		}
		if i == 0 {
			shared = ids
			continue
		}
		next := map[string]struct{}{}
		for id := range shared {
			if _, ok := ids[id]; ok {
				next[id] = struct{}{}
			}
		}
		shared = next
	}
	v1Paths := map[string]any{}
	v2Paths := map[string]any{}
	v1Schemas := map[string]any{}
	v2Schemas := map[string]any{}
	var v1Info, v2Info []string
	for _, item := range items {
		var own []string
		for _, id := range item.chunks {
			if _, ok := shared[id]; !ok {
				own = append(own, id)
			}
		}
		if len(own) == 0 && len(item.chunks) > 0 {
			own = item.chunks[len(item.chunks)-1:]
		}
		var text string
		name := item.source
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		for i := len(own) - 1; i >= 0; i-- {
			body, err := get(base + "assets/js/" + files[own[i]])
			if err != nil {
				return nil, nil, err
			}
			if strings.Contains(body, "JSON.parse") || strings.Contains(body, name) {
				text = body
				break
			}
		}
		if text == "" {
			return nil, nil, fmt.Errorf("docscheck: no chunk for %s", item.source)
		}
		objs := jsonParses(text)
		meta := map[string]any{}
		var params any
		var responses any
		var bodyWrap map[string]any
		for _, obj := range objs {
			dict, ok := obj.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := dict["permalink"]; ok {
				meta = dict
			}
			if value, ok := dict["parameters"]; ok {
				params = value
			}
			if value, ok := dict["responses"]; ok {
				responses = value
			}
			if _, ok := dict["body"]; ok {
				if _, hasLink := dict["permalink"]; !hasLink {
					bodyWrap = dict
				}
			}
		}
		var requestBody any
		if bodyWrap != nil {
			if inner, ok := bodyWrap["body"].(map[string]any); ok {
				if _, ok := inner["content"]; ok {
					requestBody = inner
				} else {
					requestBody = map[string]any{
						"required": true,
						"content":  map[string]any{"application/json": map[string]any{"schema": inner}},
					}
				}
			}
		}
		isV2 := strings.Contains(item.source, "/docs/v2/")
		title := strings.TrimSpace(stringField(meta, "title"))
		if title == "" {
			title = name
		}
		if method, path, ok := endpoint(text); ok {
			opID := stringField(meta, "id")
			if opID == "" {
				opID = title
			}
			op := map[string]any{
				"operationId": nonID.ReplaceAllString(opID, "_"),
				"summary":     title,
			}
			desc := stringField(meta, "description")
			if desc != "" && desc != title {
				op["description"] = desc
			}
			if params != nil {
				op["parameters"] = params
			}
			if requestBody != nil {
				op["requestBody"] = requestBody
			}
			if responses == nil {
				responses = map[string]any{"default": map[string]any{"description": "See docs page"}}
			}
			op["responses"] = responses
			dest := v1Paths
			if isV2 {
				dest = v2Paths
			}
			methods, _ := dest[path].(map[string]any)
			if methods == nil {
				methods = map[string]any{}
				dest[path] = methods
			}
			methods[method] = op
			continue
		}
		var schema map[string]any
		for _, obj := range objs {
			dict, ok := obj.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := dict["properties"]; ok || dict["schema"] != nil || dict["type"] != nil {
				schema = dict
				break
			}
		}
		if schema != nil {
			if inner, ok := schema["schema"].(map[string]any); ok {
				if _, hasProps := schema["properties"]; !hasProps {
					schema = inner
				}
			}
			schemaName := stringField(schema, "title")
			if schemaName == "" {
				schemaName = title
			}
			if isV2 {
				v2Schemas[schemaName] = schema
			} else {
				v1Schemas[schemaName] = schema
			}
			continue
		}
		if strings.HasSuffix(item.source, ".info.mdx") {
			line := stringField(meta, "description")
			if line == "" {
				line = stringField(meta, "title")
			}
			if isV2 {
				v2Info = append(v2Info, line)
			} else {
				v1Info = append(v1Info, line)
			}
		}
	}
	v1 := spec("GSellers Seller API v1", v1Paths, v1Schemas, v1Info, today,
		"V1 does not use the Authorization header. POST /api_sellers/api/apilogin with seller_id, timestamp and sign = SHA256(api_key + timestamp). Other methods take the returned token as the query parameter token.",
		false)
	v2 := spec("GSellers Seller API v2", v2Paths, v2Schemas, v2Info, today,
		"V2 sends the seller API key in the Authorization header with no Bearer prefix. Permissions all or api_sellers/v2 cover this contour.",
		true)
	return v1, v2, nil
}

func spec(title string, paths, schemas map[string]any, intro []string, today, auth string, v2 bool) map[string]any {
	var lines []string
	for _, line := range intro {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	description := "Snapshot of the public GSellers docs at https://seller.ggsel.com/docs, " +
		"reconstructed on " + today + " from the Docusaurus build (docusaurus-plugin-openapi-docs). " +
		"The site is generated from swaggers/swagger-seller.yaml and swaggers/swagger-seller-v2.yaml. " +
		"Those yaml files are not published. Base URL is the host used by the documentation code samples. " +
		auth
	if len(lines) > 0 {
		description += "\n\n" + strings.Join(lines, "\n\n")
	}
	out := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       title,
			"version":     "snapshot-" + today,
			"description": description,
		},
		"servers":    []any{map[string]any{"url": "https://seller.ggsel.com"}},
		"paths":      paths,
		"components": map[string]any{},
	}
	comps := out["components"].(map[string]any)
	if len(schemas) > 0 {
		comps["schemas"] = schemas
	}
	if v2 {
		out["security"] = []any{map[string]any{"apiKey": []any{}}}
		comps["securitySchemes"] = map[string]any{
			"apiKey": map[string]any{"type": "apiKey", "in": "header", "name": "Authorization"},
		}
	}
	return out
}

func contractOf(doc map[string]any) map[string]any {
	comps, _ := doc["components"].(map[string]any)
	var schemas, schemes any
	if comps != nil {
		schemas = comps["schemas"]
		schemes = comps["securitySchemes"]
	}
	return map[string]any{
		"servers":         doc["servers"],
		"security":        doc["security"],
		"paths":           doc["paths"],
		"schemas":         schemas,
		"securitySchemes": schemes,
	}
}

func operations(doc map[string]any) int {
	paths, _ := doc["paths"].(map[string]any)
	n := 0
	for _, item := range paths {
		methods, _ := item.(map[string]any)
		n += len(methods)
	}
	return n
}

var (
	nonID       = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	runtimeMain = regexp.MustCompile(`https://[^"'[:space:]\>]+/runtime~main\.[a-f0-9]+\.js`)
	mainBundle  = regexp.MustCompile(`https://[^"'[:space:]\>]+/main\.[a-f0-9]+\.js`)
	chunkBase   = regexp.MustCompile(`n\.p="(https://[^"]+)"`)
	pairRE      = regexp.MustCompile(`(\d+):"([^"]+)"`)
	routeRE     = regexp.MustCompile(`Promise\.all\(\[([^\]]+)\]\)\.then\(n\.bind\(n,\d+\)\),"(@site/docs/[^"]+)"`)
	idRE        = regexp.MustCompile(`"(\d+)"`)
	endpointRE  = regexp.MustCompile(`method:"([a-z]+)",path:"([^"]+)",context:"endpoint"`)
	parseRE     = regexp.MustCompile(`JSON\.parse\('((?:\\'|[^'])*)'\)`)
	httpClient  = &http.Client{Timeout: 60 * time.Second}
)

type route struct {
	source string
	chunks []string
}

func get(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("docscheck: request: %w", err)
	}
	req.Header.Set("User-Agent", "ggsel-docs-snapshot")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("docscheck: get %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return "", fmt.Errorf("docscheck: read %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("docscheck: get %s: status %d", url, resp.StatusCode)
	}
	return string(body), nil
}

func bundleURLs(html string) (string, string, error) {
	runtime := runtimeMain.FindString(html)
	mainJS := mainBundle.FindString(html)
	if runtime == "" || mainJS == "" {
		return "", "", fmt.Errorf("docscheck: docs page has no runtime~main.js or main.js")
	}
	return runtime, mainJS, nil
}

func chunkFiles(runtime string) (string, map[string]string, error) {
	base := chunkBase.FindStringSubmatch(runtime)
	if base == nil {
		return "", nil, fmt.Errorf("docscheck: runtime bundle has no n.p base")
	}
	marker := strings.Index(runtime, `.u=e=>"assets/js/"+`)
	if marker < 0 {
		return "", nil, fmt.Errorf("docscheck: runtime bundle has no chunk loader")
	}
	blob := runtime[marker:]
	objs := make([]string, 0, 2)
	pos := 0
	for range 2 {
		start := strings.Index(blob[pos:], "{")
		if start < 0 {
			return "", nil, fmt.Errorf("docscheck: runtime bundle chunk map is short")
		}
		start += pos
		depth := 0
		end := -1
		for j := start; j < len(blob); j++ {
			switch blob[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = j
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			return "", nil, fmt.Errorf("docscheck: runtime bundle chunk map is truncated")
		}
		objs = append(objs, blob[start:end+1])
		pos = end + 1
	}
	names := pairMap(objs[0])
	hashes := pairMap(objs[1])
	files := make(map[string]string, len(hashes))
	for id, digest := range hashes {
		name := names[id]
		if name == "" {
			name = id
		}
		files[id] = name + "." + digest + ".js"
	}
	return base[1], files, nil
}

func pairMap(s string) map[string]string {
	out := map[string]string{}
	for _, match := range pairRE.FindAllStringSubmatch(s, -1) {
		out[match[1]] = match[2]
	}
	return out
}

func routes(mainJS string) ([]route, error) {
	var found []route
	for _, match := range routeRE.FindAllStringSubmatch(mainJS, -1) {
		var chunks []string
		for _, id := range idRE.FindAllStringSubmatch(match[1], -1) {
			chunks = append(chunks, id[1])
		}
		found = append(found, route{source: match[2], chunks: chunks})
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("docscheck: main bundle has no @site/docs routes")
	}
	return found, nil
}

func endpoint(text string) (string, string, bool) {
	match := endpointRE.FindStringSubmatch(text)
	if match == nil {
		return "", "", false
	}
	return match[1], match[2], true
}

func jsonParses(text string) []any {
	var objs []any
	for _, match := range parseRE.FindAllStringSubmatch(text, -1) {
		raw := jsUnescape(match[1])
		var obj any
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			continue
		}
		objs = append(objs, obj)
	}
	return objs
}

func jsUnescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case '\\', '\'', '"', '/':
			b.WriteByte(s[i+1])
			i++
		case 'u':
			if i+5 >= len(s) {
				b.WriteByte(s[i])
				continue
			}
			r, ok := parseHex4(s[i+2 : i+6])
			if !ok {
				b.WriteByte(s[i])
				continue
			}
			if utf16.IsSurrogate(r) {
				if i+12 <= len(s) && s[i+6] == '\\' && s[i+7] == 'u' {
					if r2, ok2 := parseHex4(s[i+8 : i+12]); ok2 {
						pair := utf16.DecodeRune(r, r2)
						if pair != utf8.RuneError {
							b.WriteRune(pair)
							i += 11
							continue
						}
					}
				}
				b.WriteRune(utf8.RuneError)
				i += 5
				continue
			}
			b.WriteRune(r)
			i += 5
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

func parseHex4(s string) (rune, bool) {
	var n uint16
	for i := 0; i < 4; i++ {
		c := s[i]
		var d byte
		switch {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		default:
			return 0, false
		}
		n = n<<4 | uint16(d)
	}
	return rune(n), true
}

func stringField(m map[string]any, key string) string {
	value, _ := m[key].(string)
	return value
}

func canonical(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprint(v))
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return raw
	}
	return buf.Bytes()
}

func unified(name string, current, live []byte) string {
	a := strings.Split(string(current), "\n")
	b := strings.Split(string(live), "\n")
	var out []string
	out = append(out, "--- "+name+".committed", "+++ "+name+".live")
	for _, edit := range lineEdits(a, b) {
		out = append(out, string(edit.op)+edit.text)
	}
	return strings.Join(out, "\n")
}

func readDoc(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("docscheck: read %s: %w", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("docscheck: parse %s: %w", path, err)
	}
	return doc, nil
}

func writeDoc(path string, doc map[string]any) error {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("docscheck: encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("docscheck: write %s: %w", path, err)
	}
	return nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("docscheck: go.mod not found")
		}
		dir = parent
	}
}
