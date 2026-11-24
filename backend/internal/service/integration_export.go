package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces the controlled-export half of IntegrationService:
// column/field authorization, artifact generation sealed at rest, digest-
// verified downloads with audit, and delivery disclosures.

// exportColumns is the frozen export column whitelist.
var exportColumns = map[string]bool{
	"partyReference": true, "name": true, "phone": true, "email": true,
	"companionCount": true, "requestedSize": true, "approvedSize": true,
	"netAdmitted": true, "remainingCount": true, "status": true,
	"sourceType": true, "createdAt": true,
}

// exportAccess requires the organizer grant to carry EXPORT and every
// personal column to be inside the caller's allowed fields.
func (svc *Service) exportAccess(ctx context.Context, sc *store.Scope, c *Req, event string, columns []any) error {
	if err := c.Organizer(ctx, event); err != nil {
		return err
	}
	grant, err := svc.S.OrganizerGrantForExport(ctx, sc, event, store.StrOrEmpty(c.ActorID()))
	if err != nil {
		return err
	}
	if err := errcode.Require(containsStr(store.List(grant["allowed_actions"]), "EXPORT"),
		403, "EXPORT_FORBIDDEN", "未获得导出权限"); err != nil {
		return err
	}
	fields, err := c.Fields(ctx, event)
	if err != nil {
		return err
	}
	for _, col := range columns {
		x := store.StrOrEmpty(col)
		if err := errcode.Require(exportColumns[x], 422, "INVALID_EXPORT_COLUMN", "导出列不在允许范围"); err != nil {
			return err
		}
		if x == "name" || x == "phone" || x == "email" || x == "companionCount" {
			if err := errcode.Require(fields[x], 403, "FIELD_FORBIDDEN", "当前授权不包含所选字段"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc *Service) exportCreate(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	ev := store.StrOrEmpty(e["id"])
	columns := store.List(r["columns"])
	if err := svc.exportAccess(ctx, sc, c, ev, columns); err != nil {
		return nil, err
	}
	id := svc.S.ID()
	due, err := store.Time(e["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	expires := minTime(svc.S.Now().Add(24*time.Hour), due)
	j, err := svc.newJob(ctx, sc, ev, "EXPORT", "export:"+id, "RUNNING", store.Map(), nil, nil, expires)
	if err != nil {
		return nil, err
	}
	key := id + ".artifact"
	data := []map[string]any{}
	filter := store.Parse(r["filter"])
	parties, err := svc.S.ListPartiesAsc(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	for _, p := range parties {
		if !matchesExportFilter(p, filter) {
			continue
		}
		f, err := svc.S.Unseal(p["fields_cipher"])
		if err != nil {
			return nil, err
		}
		data = append(data, store.Map(
			"partyReference", p["party_reference"],
			"name", f["name"],
			"phone", f["phone"],
			"email", f["email"],
			"companionCount", f["companionCount"],
			"requestedSize", p["requested_size"],
			"approvedSize", p["approved_size"],
			"netAdmitted", p["net_admitted"],
			"remainingCount", store.Num(p["approved_size"])-store.Num(p["net_admitted"]),
			"status", p["status"],
			"sourceType", p["source_type"],
			"createdAt", p["created_at"],
		))
	}
	columnNames := make([]string, 0, len(columns))
	for _, col := range columns {
		columnNames = append(columnNames, store.StrOrEmpty(col))
	}
	artifact, err := svc.renderExport(store.StrOrEmpty(r["format"]), columnNames, data)
	if err != nil {
		if ec := errcode.AsError(err); ec != nil {
			return nil, ec
		}
		return nil, errcode.New(503, "EXPORT_FAILED", "导出文件生成失败")
	}
	storage, err := filepath.Abs(svc.Cfg.StorageDirectory)
	if err != nil {
		return nil, fmt.Errorf("storage directory: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString(artifact)
	sealed, err := svc.S.Seal(sc, store.Map("base64", b64))
	if err != nil {
		return nil, err
	}
	if err := writeManagedArtifact(storage, key, sealed); err != nil {
		return nil, errcode.New(503, "EXPORT_FAILED", "导出文件生成失败")
	}
	columnsJSON, err := store.Stringify(columns)
	if err != nil {
		return nil, err
	}
	filterJSON, err := store.Stringify(filter)
	if err != nil {
		return nil, err
	}
	purposeCipher, err := svc.S.Seal(sc, store.Map("purpose", r["purpose"]))
	if err != nil {
		return nil, err
	}
	now := svc.S.Now()
	if err := svc.S.InsertExport(ctx, sc, id, ev, store.StrOrEmpty(c.ActorID()), store.StrOrEmpty(j["id"]),
		store.StrOrEmpty(r["format"]), columnsJSON, filterJSON, purposeCipher, key,
		svc.S.Crypto.Hash(b64), len(data), "SUCCEEDED", expires, now); err != nil {
		return nil, err
	}
	if err := svc.S.CompleteJob(ctx, sc, id, now, store.StrOrEmpty(j["id"])); err != nil {
		return nil, err
	}
	x, err := svc.S.ExportByID(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	return svc.exportView(x), nil
}

// matchesExportFilter applies the frozen export filter (status/sourceType/
// progress equality and inclusive created bounds).
func matchesExportFilter(p map[string]any, f map[string]any) bool {
	for _, k := range []string{"status", "sourceType"} {
		if v, has := f[k]; has {
			column := k
			if k == "sourceType" {
				column = "source_type"
			}
			if store.StrOrEmpty(v) != store.StrOrEmpty(p[column]) {
				return false
			}
		}
	}
	if v, has := f["progress"]; has {
		n, a := store.Num(p["net_admitted"]), store.Num(p["approved_size"])
		progress := "COMPLETE"
		if n == 0 {
			progress = "NONE"
		} else if n < a {
			progress = "PARTIAL"
		}
		if progress != store.StrOrEmpty(v) {
			return false
		}
	}
	at, err := store.Time(p["created_at"])
	if err != nil {
		return false
	}
	if v, has := f["createdFrom"]; has {
		from, err := store.Time(v)
		if err != nil || at.Before(from) {
			return false
		}
	}
	if v, has := f["createdTo"]; has {
		to, err := store.Time(v)
		if err != nil || at.After(to) {
			return false
		}
	}
	return true
}

// renderExport produces the CSV (BOM/CRLF) or XLSX binary.
func (svc *Service) renderExport(format string, columns []string, data []map[string]any) ([]byte, error) {
	if format == "CSV" {
		return renderCSVExport(columns, data), nil
	}
	rows := make([][]string, 0, len(data))
	for _, row := range data {
		cells := make([]string, len(columns))
		for i, col := range columns {
			cells[i] = safeCell(row[col])
		}
		rows = append(rows, cells)
	}
	return renderXLSXExport(columns, rows)
}

// writeManagedArtifact creates the sealed artifact exclusively (CREATE_NEW).
func writeManagedArtifact(storage, key string, data []byte) error {
	if err := os.MkdirAll(storage, 0o755); err != nil {
		return fmt.Errorf("storage mkdir: %w", err)
	}
	p := filepath.Clean(filepath.Join(storage, key))
	if !strings.HasPrefix(p, storage+string(os.PathSeparator)) {
		return fmt.Errorf("invalid managed artifact")
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("artifact create: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("artifact write: %w", err)
	}
	return nil
}

// exportRow loads an export and re-checks ownership plus column authorization.
func (svc *Service) exportRow(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (map[string]any, error) {
	x, err := svc.S.ExportByIDEvent(ctx, sc, store.StrOrEmpty(r["exportId"]), store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(c.ActorID()) == store.StrOrEmpty(x["actor_id"]),
		403, "EXPORT_FORBIDDEN", "只能访问本人生成的导出"); err != nil {
		return nil, err
	}
	if err := svc.exportAccess(ctx, sc, c, store.StrOrEmpty(r["eventId"]), store.List(x["columns_json"])); err != nil {
		return nil, err
	}
	return x, nil
}

func (svc *Service) exportView(x map[string]any) map[string]any {
	status := store.StrOrEmpty(x["status"])
	if x["expires_at"] != nil {
		if expires, err := store.Time(x["expires_at"]); err == nil && !expires.After(svc.S.Now()) {
			status = "EXPIRED"
		}
	}
	return store.Map(
		"exportId", x["id"],
		"eventId", x["event_id"],
		"status", svc.value("JobStatus", status),
		"format", svc.value("ExportFormat", x["format"]),
		"columns", store.List(x["columns_json"]),
		"rowCount", x["row_count"],
		"expiresAt", x["expires_at"],
		"version", x["version"],
	)
}

func (svc *Service) exportDownload(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
		return nil, err
	}
	x, err := svc.exportRow(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	expires, err := store.Time(x["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("export expiry: %w", err)
	}
	if err := errcode.Require(expires.After(svc.S.Now()), 410, "EXPORT_EXPIRED", "导出文件已过期"); err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(x["status"]) == "SUCCEEDED",
		409, "EXPORT_NOT_READY", "导出文件尚未生成"); err != nil {
		return nil, err
	}
	storage, err := filepath.Abs(svc.Cfg.StorageDirectory)
	if err != nil {
		return nil, fmt.Errorf("storage directory: %w", err)
	}
	p := filepath.Clean(filepath.Join(storage, store.StrOrEmpty(x["artifact_key"])))
	if err := errcode.Require(p == storage || strings.HasPrefix(p, storage+string(os.PathSeparator)),
		403, "FORBIDDEN", "无权访问文件"); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, errcode.New(410, "EXPORT_EXPIRED", "导出文件不可用，请重新生成")
	}
	payload, err := svc.S.Unseal(data)
	if err != nil {
		return nil, errcode.New(410, "EXPORT_EXPIRED", "导出文件不可用，请重新生成")
	}
	artifact, err := base64.StdEncoding.DecodeString(store.StrOrEmpty(payload["base64"]))
	if err != nil {
		return nil, errcode.New(410, "EXPORT_EXPIRED", "导出文件不可用，请重新生成")
	}
	if !bytes.Equal(x["artifact_digest"].([]byte), svc.S.Crypto.Hash(base64.StdEncoding.EncodeToString(artifact))) {
		return nil, errcode.New(410, "EXPORT_EXPIRED", "导出文件校验失败")
	}
	var deviceID any
	if c.Device != nil {
		deviceID = c.Device["id"]
	}
	if err := svc.S.Audit(ctx, sc, store.StrOrEmpty(r["eventId"]), "ExportDownload", c.ActorID(), deviceID, x["id"], nil, nil); err != nil {
		return nil, err
	}
	csv := store.StrOrEmpty(x["format"]) == "CSV"
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	filename := "unigate.xlsx"
	if csv {
		contentType = "text/csv;charset=UTF-8"
		filename = "unigate.csv"
	}
	return Binary{Data: artifact, ContentType: contentType, Filename: filename}, nil
}

func (svc *Service) deliveryList(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
		return nil, err
	}
	rows, err := svc.S.ListDeliveries(ctx, sc, store.StrOrEmpty(r["eventId"]))
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(rows))
	for _, x := range rows {
		v, err := svc.deliveryView(x)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return store.Page(out, r), nil
}

func (svc *Service) deliveryRecord(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
		return nil, err
	}
	x, err := svc.exportRow(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	cols := store.List(r["columns"])
	allowed := map[string]bool{}
	for _, col := range store.List(x["columns_json"]) {
		allowed[store.StrOrEmpty(col)] = true
	}
	inScope := len(cols) > 0
	for _, col := range cols {
		if !allowed[store.StrOrEmpty(col)] {
			inScope = false
		}
	}
	if err := errcode.Require(inScope, 422, "DELIVERY_SCOPE_INVALID", "交付范围必须属于原导出列"); err != nil {
		return nil, err
	}
	expires, err := store.Time(x["expires_at"])
	if err != nil {
		return nil, fmt.Errorf("export expiry: %w", err)
	}
	if err := errcode.Require(store.StrOrEmpty(x["status"]) == "SUCCEEDED" && expires.After(svc.S.Now()),
		410, "EXPORT_EXPIRED", "原导出不可用"); err != nil {
		return nil, err
	}
	id := svc.S.ID()
	recipientCipher, err := svc.S.Seal(sc, store.Map("value", r["recipient"]))
	if err != nil {
		return nil, err
	}
	purposeCipher, err := svc.S.Seal(sc, store.Map("value", r["purpose"]))
	if err != nil {
		return nil, err
	}
	retentionCipher, err := svc.S.Seal(sc, store.Map("value", r["retentionNote"]))
	if err != nil {
		return nil, err
	}
	columnsJSON, err := store.Stringify(cols)
	if err != nil {
		return nil, err
	}
	if err := svc.S.InsertDelivery(ctx, sc, id, store.StrOrEmpty(r["eventId"]), store.StrOrEmpty(x["id"]),
		recipientCipher, purposeCipher, columnsJSON, retentionCipher, store.StrOrEmpty(c.ActorID()), svc.S.Now()); err != nil {
		return nil, err
	}
	row, err := svc.S.DeliveryByID(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	return svc.deliveryView(row)
}

func (svc *Service) deliveryView(x map[string]any) (map[string]any, error) {
	recipient, err := svc.S.Unseal(x["recipient_cipher"])
	if err != nil {
		return nil, err
	}
	purpose, err := svc.S.Unseal(x["purpose_cipher"])
	if err != nil {
		return nil, err
	}
	retention, err := svc.S.Unseal(x["retention_note_cipher"])
	if err != nil {
		return nil, err
	}
	return store.Map(
		"deliveryId", x["id"],
		"eventId", x["event_id"],
		"exportId", x["export_id"],
		"recipient", recipient["value"],
		"purpose", purpose["value"],
		"retentionNote", retention["value"],
		"columns", store.List(x["columns_json"]),
		"createdAt", x["created_at"],
		"version", x["version"],
	), nil
}
