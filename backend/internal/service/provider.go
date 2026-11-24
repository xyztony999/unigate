package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// providerSetting reads unigate.providers.connections.<id>.<name>.
func (svc *Service) providerSetting(connection map[string]any, name string) string {
	return svc.Cfg.Providers.Setting(store.StrOrEmpty(connection["id"]), name)
}

// providerToken resolves a usable platform access token.
func (svc *Service) providerToken(connection map[string]any) (string, error) {
	creds, err := svc.S.Unseal(connection["credentials_cipher"])
	if err != nil {
		return "", err
	}
	token := store.StrOrEmpty(creds["accessToken"])
	if strings.TrimSpace(token) == "" {
		token = svc.providerSetting(connection, "access-token")
	}
	status := store.StrOrEmpty(connection["status"])
	expValid := true
	if connection["expires_at"] != nil {
		exp, perr := store.Time(connection["expires_at"])
		expValid = perr == nil && exp.After(svc.S.Now())
	}
	if strings.TrimSpace(token) == "" || status == "REVOKED" || status == "EXPIRED" || !expValid {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "连接未配置有效的平台令牌")
	}
	return token, nil
}

// providerRedirect validates the registered HTTPS callback address.
func (svc *Service) providerRedirect(c map[string]any) (string, error) {
	raw := svc.providerSetting(c, "redirect-uri")
	u, err := url.Parse(raw)
	if err != nil || raw == "" {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "授权回调地址未配置")
	}
	if u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Path != "/api/v1/integration/connection/callback" {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "需配置已向提供方登记的HTTPS回调地址")
	}
	return raw, nil
}

// providerAuthorizeURL builds the OAuth authorization URL.
func (svc *Service) providerAuthorizeURL(c map[string]any, state string) (string, error) {
	provider := store.StrOrEmpty(c["provider"])
	creds, err := svc.S.Unseal(c["credentials_cipher"])
	if err != nil {
		return "", err
	}
	if (provider != "FEISHU" && provider != "TENCENT") || c["client_id"] == nil {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "请先配置提供方应用凭据")
	}
	if _, has := creds["clientSecret"]; !has {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "请先配置提供方应用凭据")
	}
	scope := "all"
	if provider == "FEISHU" {
		scope = svc.providerSetting(c, "scope")
	}
	if strings.TrimSpace(scope) == "" {
		return "", errcode.New(503, "PROVIDER_UNAVAILABLE", "请按已审核权限配置授权范围")
	}
	redirect, err := svc.providerRedirect(c)
	if err != nil {
		return "", err
	}
	base := "https://docs.qq.com/oauth/v2/authorize"
	if provider == "FEISHU" {
		base = "https://accounts.feishu.cn/open-apis/authen/v1/authorize"
	}
	out := base + "?client_id=" + url.QueryEscape(store.StrOrEmpty(c["client_id"])) +
		"&redirect_uri=" + url.QueryEscape(redirect) +
		"&response_type=code&scope=" + url.QueryEscape(scope) +
		"&state=" + url.QueryEscape(state)
	if provider == "TENCENT" {
		out += "&new_login=1"
	}
	return out, nil
}

// providerExchange swaps an OAuth code for tokens.
func (svc *Service) providerExchange(ctx context.Context, c map[string]any, code string) (map[string]any, error) {
	provider := store.StrOrEmpty(c["provider"])
	creds, err := svc.S.Unseal(c["credentials_cipher"])
	if err != nil {
		return nil, err
	}
	redirect, err := svc.providerRedirect(c)
	if err != nil {
		return nil, err
	}
	body := store.Map("grant_type", "authorization_code", "client_id", c["client_id"], "client_secret", creds["clientSecret"], "redirect_uri", redirect, "code", code)
	var req *http.Request
	if provider == "FEISHU" {
		bodyJSON, err := store.Stringify(body)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequestWithContext(ctx, "POST", "https://open.feishu.cn/open-apis/authen/v2/oauth/token", strings.NewReader(bodyJSON))
		if err != nil {
			return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "提供方授权失败，请重新授权")
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		q := url.Values{}
		for k, v := range body {
			q.Set(k, store.StrOrEmpty(v))
		}
		req, err = http.NewRequestWithContext(ctx, "GET", "https://docs.qq.com/oauth/v2/token?"+q.Encode(), nil)
		if err != nil {
			return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "提供方授权失败，请重新授权")
		}
	}
	result, err := svc.providerDo(req, 65536, 15*time.Second)
	if err != nil {
		if ec := errcode.AsError(err); ec != nil {
			return nil, ec
		}
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "提供方授权失败，请重新授权")
	}
	if result["access_token"] == nil || store.Num(result["expires_in"]) <= 0 {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "提供方授权未成功")
	}
	return store.Map("accessToken", result["access_token"], "refreshToken", result["refresh_token"], "openId", result["user_id"], "expiresIn", result["expires_in"]), nil
}

func (svc *Service) providerResources(connection map[string]any) ([]string, error) {
	raw := svc.providerSetting(connection, "resources")
	if strings.TrimSpace(raw) == "" {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "尚未配置经过授权的资源清单")
	}
	out := []string{}
	for _, x := range strings.Split(raw, ",") {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out, nil
}

// providerList lists authorized resources (metadata only).
func (svc *Service) providerList(ctx context.Context, connection map[string]any) ([]any, error) {
	if _, err := svc.providerToken(connection); err != nil {
		return nil, err
	}
	resources, err := svc.providerResources(connection)
	if err != nil {
		return nil, err
	}
	result := []any{}
	for _, resource := range resources {
		if _, err := svc.providerFields(ctx, connection, resource); err != nil {
			return nil, err
		}
		kind := "SPREADSHEET"
		if store.StrOrEmpty(connection["provider"]) == "FEISHU" {
			kind = "BITABLE"
		}
		result = append(result, store.Map("resourceId", resource, "title", resource, "kind", kind, "readable", true))
	}
	return result, nil
}

var resourcePartRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,150}$`)

func (svc *Service) providerResourceParts(c map[string]any, resource string) ([]string, error) {
	resources, err := svc.providerResources(c)
	if err != nil {
		return nil, err
	}
	found := false
	for _, x := range resources {
		if x == resource {
			found = true
			break
		}
	}
	if !found {
		return nil, errcode.New(403, "RESOURCE_FORBIDDEN", "资源不在当前连接授权范围内")
	}
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return nil, errcode.New(422, "RESOURCE_ID_INVALID", "资源标识格式应为appToken/tableId或fileId/sheetId")
	}
	for _, p := range parts {
		if !resourcePartRe.MatchString(p) {
			return nil, errcode.New(422, "RESOURCE_ID_INVALID", "资源标识格式应为appToken/tableId或fileId/sheetId")
		}
	}
	return parts, nil
}

// providerFields reads column metadata for a resource.
func (svc *Service) providerFields(ctx context.Context, c map[string]any, resource string) ([]any, error) {
	p, err := svc.providerResourceParts(c, resource)
	if err != nil {
		return nil, err
	}
	provider := store.StrOrEmpty(c["provider"])
	if provider == "FEISHU" {
		out := []any{}
		cursor := ""
		for {
			u := "https://open.feishu.cn/open-apis/bitable/v1/apps/" + p[0] + "/tables/" + p[1] + "/fields?page_size=100"
			if cursor != "" {
				u += "&page_token=" + url.QueryEscape(cursor)
			}
			result, err := svc.providerGet(ctx, c, u)
			if err != nil {
				return nil, err
			}
			data := store.Parse(result["data"])
			for _, x := range store.List(data["items"]) {
				field := store.Parse(x)
				name := store.StrOrEmpty(field["field_name"])
				out = append(out, store.Map("code", name, "label", name))
			}
			if data["has_more"] == true {
				cursor = store.StrOrEmpty(data["page_token"])
			} else {
				cursor = ""
			}
			if len(out) > 200 {
				return nil, errcode.New(422, "PROVIDER_LIMIT", "来源最多支持200列")
			}
			if cursor == "" {
				break
			}
		}
		return out, nil
	}
	if provider == "TENCENT" {
		rows, err := svc.tencentRows(ctx, c, p, "A1:GR1")
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return nil, errcode.New(422, "PROVIDER_DATA_INVALID", "来源缺少表头")
		}
		headers := rows[0]
		seen := map[string]bool{}
		for _, x := range headers {
			if strings.TrimSpace(x) == "" || seen[x] {
				return nil, errcode.New(422, "PROVIDER_DATA_INVALID", "来源表头须唯一非空")
			}
			seen[x] = true
		}
		out := []any{}
		for _, x := range headers {
			out = append(out, store.Map("code", x, "label", x))
		}
		return out, nil
	}
	return nil, errcode.New(422, "INVALID_PROVIDER", "此连接不支持名单读取")
}

// RemoteRow is a pulled source record with a stable id.
type RemoteRow struct {
	ID     string
	Fields map[string]any
}

// providerPull reads all records of a resource.
func (svc *Service) providerPull(ctx context.Context, c map[string]any, resource string) ([]RemoteRow, error) {
	p, err := svc.providerResourceParts(c, resource)
	if err != nil {
		return nil, err
	}
	out := []RemoteRow{}
	provider := store.StrOrEmpty(c["provider"])
	if provider == "FEISHU" {
		cursor := ""
		for {
			u := "https://open.feishu.cn/open-apis/bitable/v1/apps/" + p[0] + "/tables/" + p[1] + "/records?page_size=500"
			if cursor != "" {
				u += "&page_token=" + url.QueryEscape(cursor)
			}
			result, err := svc.providerGet(ctx, c, u)
			if err != nil {
				return nil, err
			}
			data := store.Parse(result["data"])
			for _, x := range store.List(data["items"]) {
				record := store.Parse(x)
				normalized := map[string]any{}
				for k, v := range store.Parse(record["fields"]) {
					pv, err := providerPlain(v)
					if err != nil {
						return nil, err
					}
					normalized[k] = pv
				}
				out = append(out, RemoteRow{ID: store.StrOrEmpty(record["record_id"]), Fields: normalized})
			}
			if data["has_more"] == true {
				cursor = store.StrOrEmpty(data["page_token"])
			} else {
				cursor = ""
			}
			if len(out) > 10000 {
				return nil, errcode.New(422, "PROVIDER_LIMIT", "来源最多支持10000行")
			}
			if cursor == "" {
				break
			}
		}
	} else if provider == "TENCENT" {
		fieldList, err := svc.providerFields(ctx, c, resource)
		if err != nil {
			return nil, err
		}
		headers := []string{}
		for _, x := range fieldList {
			headers = append(headers, store.StrOrEmpty(x.(map[string]any)["code"]))
		}
		keyColumn := svc.providerSetting(c, "record-id-column")
		hasKey := false
		for _, h := range headers {
			if h == keyColumn {
				hasKey = true
			}
		}
		if !hasKey {
			return nil, errcode.New(422, "SOURCE_ID_REQUIRED", "腾讯名单需显式配置稳定记录ID列，不能用可移动行号识别人员")
		}
		width := len(headers)
		block := 1000
		if 10000/maxInt(width, 1) < block {
			block = 10000 / maxInt(width, 1)
		}
		last := excelColumn(width)
		finished := false
		for start := 2; start <= 10001 && !finished; start += block {
			end := start + block - 1
			if end > 10002 {
				end = 10002
			}
			data, err := svc.tencentRows(ctx, c, p, fmt.Sprintf("A%d:%s%d", start, last, end))
			if err != nil {
				return nil, err
			}
			if len(data) == 0 {
				break
			}
			for _, values := range data {
				row := map[string]any{}
				for j, h := range headers {
					if j < len(values) {
						row[h] = values[j]
					} else {
						row[h] = ""
					}
				}
				allBlank := true
				for _, v := range row {
					if strings.TrimSpace(store.StrOrEmpty(v)) != "" {
						allBlank = false
					}
				}
				if allBlank {
					finished = true
					continue
				}
				if finished {
					return nil, errcode.New(422, "PROVIDER_DATA_INVALID", "名单中间存在空行，请整理连续范围")
				}
				id := store.StrOrEmpty(row[keyColumn])
				if strings.TrimSpace(id) == "" {
					return nil, errcode.New(422, "SOURCE_ID_REQUIRED", "来源记录ID不能为空")
				}
				out = append(out, RemoteRow{ID: id, Fields: row})
				if len(out) > 10000 {
					return nil, errcode.New(422, "PROVIDER_LIMIT", "来源最多支持10000行")
				}
			}
			if len(data) < end-start+1 {
				break
			}
		}
	} else {
		return nil, errcode.New(422, "INVALID_PROVIDER", "此连接不支持名单读取")
	}
	seen := map[string]bool{}
	for _, r := range out {
		if seen[r.ID] {
			return nil, errcode.New(422, "SOURCE_ID_CONFLICT", "来源记录ID必须唯一")
		}
		seen[r.ID] = true
	}
	return out, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// providerPlain flattens a Feishu field value to text (numbers normalized).
func providerPlain(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch t := v.(type) {
	case json.Number:
		return plainNumber(t.String()), nil
	case float64:
		return plainNumber(fmt.Sprintf("%v", t)), nil
	case int:
		return plainNumber(fmt.Sprintf("%d", t)), nil
	case int64:
		return plainNumber(fmt.Sprintf("%d", t)), nil
	case string:
		return t, nil
	case []any:
		var b strings.Builder
		for _, x := range t {
			if m, ok := x.(map[string]any); ok {
				if txt, has := m["text"]; has {
					b.WriteString(store.StrOrEmpty(txt))
					continue
				}
			}
			return "", errcode.New(422, "PROVIDER_DATA_INVALID", "来源包含不支持的复杂字段，请选择文本或数值字段")
		}
		return b.String(), nil
	default:
		return "", errcode.New(422, "PROVIDER_DATA_INVALID", "来源包含不支持的复杂字段")
	}
}

// plainNumber reproduces BigDecimal.stripTrailingZeros().toPlainString().
func plainNumber(s string) string {
	rat, ok := new(big.Rat).SetString(s)
	if !ok {
		return s
	}
	if rat.IsInt() {
		return rat.Num().String()
	}
	// Render the exact decimal expansion without trailing zeros.
	f := new(big.Float).SetRat(rat)
	text := f.Text('f', -1)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(text, "0")
		text = strings.TrimSuffix(text, ".")
	}
	return text
}

func (svc *Service) tencentRows(ctx context.Context, c map[string]any, p []string, rng string) ([][]string, error) {
	result, err := svc.providerGet(ctx, c, "https://docs.qq.com/openapi/spreadsheet/v3/files/"+p[0]+"/"+p[1]+"/"+rng)
	if err != nil {
		return nil, err
	}
	data := store.Parse(result["data"])
	grid := store.Parse(data["gridData"])
	rows := [][]string{}
	for _, x := range store.List(grid["rows"]) {
		values := []string{}
		for _, y := range store.List(store.Parse(x)["values"]) {
			cell := store.Parse(y)
			value := store.Parse(cell["cellValue"])
			if _, has := value["formula"]; has {
				return nil, errcode.New(422, "PROVIDER_DATA_INVALID", "来源公式单元格不可导入")
			}
			text := value["text"]
			if text == nil {
				text = value["number"]
			}
			if text == nil {
				text = ""
			}
			pv, err := providerPlain(text)
			if err != nil {
				return nil, err
			}
			values = append(values, pv)
		}
		for len(values) > 0 && strings.TrimSpace(values[len(values)-1]) == "" {
			values = values[:len(values)-1]
		}
		rows = append(rows, values)
	}
	return rows, nil
}

// providerGet performs an authenticated provider GET and validates the envelope.
func (svc *Service) providerGet(ctx context.Context, c map[string]any, u string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "暂时无法连接提供方")
	}
	req.Header.Set("Accept", "application/json")
	if store.StrOrEmpty(c["provider"]) == "FEISHU" {
		token, err := svc.providerToken(c)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		creds, err := svc.S.Unseal(c["credentials_cipher"])
		if err != nil {
			return nil, err
		}
		openID := store.StrOrEmpty(creds["openId"])
		if openID == "" {
			openID = svc.providerSetting(c, "open-id")
		}
		if strings.TrimSpace(openID) == "" || c["client_id"] == nil {
			return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "腾讯连接尚未配置应用或用户身份")
		}
		token, err := svc.providerToken(c)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Access-Token", token)
		req.Header.Set("Open-Id", openID)
		req.Header.Set("Client-Id", store.StrOrEmpty(c["client_id"]))
	}
	result, err := svc.providerDo(req, 10*1024*1024, 15*time.Second)
	if err != nil {
		return nil, err
	}
	code := result["code"]
	if code == nil {
		code = result["ret"]
	}
	if code == nil {
		code = 0
	}
	if store.Num(code) != 0 {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "提供方拒绝读取，请检查权限")
	}
	return result, nil
}

// providerDo executes a request with a size cap and no redirects.
func (svc *Service) providerDo(req *http.Request, maxBytes int64, timeout time.Duration) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := svc.HTTP.Do(req)
	if err != nil {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "暂时无法连接提供方")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "暂时无法连接提供方")
	}
	if int64(len(body)) > maxBytes {
		return nil, errcode.New(503, "PROVIDER_LIMIT", "提供方响应超出限制")
	}
	if resp.StatusCode != 200 {
		status := 503
		if resp.StatusCode == 429 {
			status = 429
		}
		return nil, errcode.New(status, "PROVIDER_UNAVAILABLE", "提供方读取失败，请检查授权状态")
	}
	var result map[string]any
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.UseNumber()
	if err := dec.Decode(&result); err != nil {
		return nil, errcode.New(503, "PROVIDER_UNAVAILABLE", "暂时无法连接提供方")
	}
	return result, nil
}

func excelColumn(n int) string {
	out := ""
	for n > 0 {
		n--
		out = string(rune('A'+n%26)) + out
		n /= 26
	}
	return out
}