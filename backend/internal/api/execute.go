package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"unigate/backend/internal/api/middleware"
	"unigate/backend/internal/contract"
	"unigate/backend/internal/errcode"
	"unigate/backend/internal/service"
	"unigate/backend/internal/store"
)

// execute is the per-operation transport shell reproducing
// ApiController.execute(): parse input -> validate request schema ->
// touchStaffSession (outside the transaction, as in Java) -> begin the
// READ COMMITTED 5s transaction -> Authenticate inside it -> Dispatch ->
// commit -> cookies + frozen envelope. Statistics passes a transaction-less
// scope so Dispatch opens its own REPEATABLE READ read-only snapshot.
func (h *handler) execute(op *contract.Operation) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		r, err := h.parseInput(c, op)
		if err != nil {
			h.fail(c, op, nil, err)
			return
		}
		info := requestInfo(c)
		if err := h.svc.TouchStaffSession(ctx, info); err != nil {
			h.fail(c, op, nil, err)
			return
		}
		sc := &store.Scope{}
		if op.Name != "Statistics" {
			begun, err := h.svc.S.Begin(ctx, store.TxOptions{Timeout: 5 * time.Second})
			if err != nil {
				h.fail(c, op, nil, fmt.Errorf("begin request tx: %w", err))
				return
			}
			sc = begun
		}
		defer h.svc.S.Rollback(ctx, sc)
		req, err := h.svc.Authenticate(ctx, sc, op, r, info)
		if err != nil {
			h.fail(c, op, sc, err)
			return
		}
		data, err := h.svc.Dispatch(ctx, sc, op.Name, r, req)
		if err != nil {
			h.fail(c, op, sc, err)
			return
		}
		// No-op for Statistics: Dispatch already committed its snapshot scope.
		if err := h.svc.S.Commit(ctx, sc); err != nil {
			h.fail(c, op, sc, fmt.Errorf("commit request tx: %w", err))
			return
		}
		writeCookies(c, sc)
		if b, ok := data.(service.Binary); ok {
			writeBinary(c, b)
			return
		}
		h.writeOK(c, op, data)
	}
}

// parseInput binds the request to the operation input map and validates it
// against the frozen request schema (ApiController.execute input branches).
func (h *handler) parseInput(c *gin.Context, op *contract.Operation) (map[string]any, error) {
	var r map[string]any
	switch {
	case op.Upload:
		parsed, err := parseUpload(c)
		if err != nil {
			return nil, err
		}
		r = parsed
	case strings.EqualFold(op.Method, http.MethodGet):
		r = make(map[string]any)
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				r[k] = v[0]
			}
		}
	default:
		dec := json.NewDecoder(c.Request.Body)
		dec.UseNumber()
		if err := dec.Decode(&r); err != nil {
			return nil, invalidRequest(err)
		}
		if r == nil {
			return nil, errcode.New(http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效")
		}
	}
	probe := r
	if op.Upload {
		// Java validates uploads with the original filename in place of the
		// file object; the raw bytes stay in r for the service layer.
		probe = make(map[string]any, len(r))
		for k, v := range r {
			if k != "file" {
				probe[k] = v
			}
		}
		if f, ok := r["file"].(service.UploadFile); ok {
			probe["file"] = f.Filename
		} else {
			probe["file"] = nil
		}
	}
	if err := h.svc.C.Validate(op.Request, probe); err != nil {
		return nil, fmt.Errorf("validate %s request: %w", op.Name, err)
	}
	return r, nil
}

// parseUpload reads the multipart form (BatchUpload). The body is already
// capped at 11MB by middleware.BodyLimit; each file is capped at 10MB, both
// surfacing the frozen ApiErrors 422 envelope.
func parseUpload(c *gin.Context) (map[string]any, error) {
	if err := c.Request.ParseMultipartForm(middleware.MaxFileBytes); err != nil {
		if middleware.IsOversize(err) {
			return nil, oversizeFile()
		}
		return nil, invalidRequest(err)
	}
	r := make(map[string]any)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			r[k] = v[0]
		}
	}
	form := c.Request.MultipartForm
	for k, v := range form.Value {
		if len(v) > 0 {
			r[k] = v[0]
		}
	}
	files := form.File["file"]
	if len(files) == 0 {
		r["file"] = nil
		return r, nil
	}
	fh := files[0]
	if fh.Size > middleware.MaxFileBytes {
		return nil, oversizeFile()
	}
	f, err := fh.Open()
	if err != nil {
		return nil, errcode.New(http.StatusUnprocessableEntity, "INVALID_FILE", "无法读取文件")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, middleware.MaxFileBytes+1))
	if err != nil {
		return nil, errcode.New(http.StatusUnprocessableEntity, "INVALID_FILE", "无法读取文件")
	}
	if len(data) > middleware.MaxFileBytes {
		return nil, oversizeFile()
	}
	r["file"] = service.UploadFile{Data: data, Filename: fh.Filename}
	return r, nil
}

func oversizeFile() error {
	return errcode.New(http.StatusUnprocessableEntity, "FIELD_VALIDATION_FAILED", "文件不得超过 10MiB")
}

// invalidRequest maps JSON/multipart parse failures to the frozen
// JsonProcessingException / IllegalArgumentException catch (400).
func invalidRequest(err error) error {
	return errcode.Wrapf(errcode.New(http.StatusBadRequest, "INVALID_REQUEST", "请求格式无效"),
		"parse request body: %v", err)
}

// requestInfo collects the transport-derived inputs Authenticate needs
// (cookies, Origin, X-CSRF-Token, X-Client-Context-ID, peer address).
func requestInfo(c *gin.Context) service.RequestInfo {
	cookies := make(map[string]string)
	for _, ck := range c.Request.Cookies() {
		cookies[ck.Name] = ck.Value
	}
	return service.RequestInfo{
		Cookies:         cookies,
		Origin:          c.GetHeader("Origin"),
		CSRF:            c.GetHeader("X-CSRF-Token"),
		ClientContextID: c.GetHeader("X-Client-Context-ID"),
		ClientIP:        c.ClientIP(),
		ServerOrigin:    serverOrigin(c.Request),
	}
}

// serverOrigin is the request's own scheme://host, used by Authenticate to
// accept same-origin POSTs alongside the configured allowed origin.
func serverOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fp := r.Header.Get("X-Forwarded-Proto"); fp != "" {
		scheme = strings.TrimSpace(strings.SplitN(fp, ",", 2)[0])
	}
	return scheme + "://" + r.Host
}

// fail writes any cookies accumulated before the failure (Java adds
// Set-Cookie headers to the response immediately, so they survive a later
// exception) and then the frozen error envelope with the real HTTP status.
func (h *handler) fail(c *gin.Context, op *contract.Operation, sc *store.Scope, err error) {
	writeCookies(c, sc)
	status, code, message := errorResponse(err)
	if status == http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed",
			"requestId", middleware.ID(c), "operation", op.Name, "error", err)
	}
	middleware.AbortWithError(c, status, code, message)
}

// errorResponse reproduces the controller catch ladder: business ApiException
// keeps its status/code/message; lock/serialization and integrity failures map
// to the frozen 409 envelopes; anything else is a 500 without internals.
func errorResponse(err error) (int, string, string) {
	if e := errcode.AsError(err); e != nil {
		return e.Status, e.Code, e.Message
	}
	if store.IsLockConflict(err) {
		return http.StatusConflict, "OPERATION_IN_PROGRESS", "操作竞争，请沿用原操作编号核对"
	}
	if store.IsIntegrityViolation(err) {
		return http.StatusConflict, "DATA_CONFLICT", "数据已变化或记录冲突，请刷新核对"
	}
	return http.StatusInternalServerError, "INTERNAL_ERROR", "服务暂不可用，请沿原操作编号核对结果"
}

// writeCookies flushes the scope's accumulated Set-Cookie headers
// (name=value; Path=/; Max-Age=43200; Secure; HttpOnly; SameSite=... built by
// the service layer, staff Lax / others Strict).
func writeCookies(c *gin.Context, sc *store.Scope) {
	if sc == nil {
		return
	}
	for _, cookie := range sc.Cookies {
		c.Writer.Header().Add("Set-Cookie", cookie)
	}
}

// writeOK writes the frozen success envelope with the operation's HTTP status
// (200, or 202 for the accepted async operations).
func (h *handler) writeOK(c *gin.Context, op *contract.Operation, data any) {
	status := op.Status
	if status == 0 {
		status = http.StatusOK
	}
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Writer.WriteHeader(status)
	body := store.Map(
		"code", "OK",
		"message", "操作成功",
		"data", store.JSONReady(data),
		"requestId", middleware.ID(c),
	)
	if err := json.NewEncoder(c.Writer).Encode(body); err != nil {
		slog.ErrorContext(c.Request.Context(), "write response envelope failed",
			"requestId", middleware.ID(c), "operation", op.Name, "error", err)
	}
}

// writeBinary streams an export artifact (ExportDownload) with the frozen
// Content-Type / Content-Disposition instead of the JSON envelope.
func writeBinary(c *gin.Context, b service.Binary) {
	name := strings.NewReplacer("\"", "", "\r", "", "\n", "").Replace(b.Filename)
	c.Writer.Header().Set("Content-Type", b.ContentType)
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.Write(b.Data); err != nil {
		slog.ErrorContext(c.Request.Context(), "write binary response failed",
			"requestId", middleware.ID(c), "error", err)
	}
}
