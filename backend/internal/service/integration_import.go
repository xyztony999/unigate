package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"unigate/backend/internal/errcode"
	"unigate/backend/internal/store"
)

// This file reproduces the import half of IntegrationService: batch upload,
// mapping preview (error rows excluded from selection), duplicate resolution
// and the atomic whole-batch commit. Any conflict rolls the entire batch back
// because every step runs inside the caller's request transaction.

func (svc *Service) batchUpload(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := svc.intakeOpen(e); err != nil {
		return nil, err
	}
	f, ok := r["file"].(UploadFile)
	if err := errcode.Require(ok, 422, "INVALID_FILE", "请选择CSV或XLSX文件"); err != nil {
		return nil, err
	}
	data := f.Data
	if err := errcode.Require(len(data) > 0 && len(data) <= 10*1024*1024,
		422, "INVALID_FILE", "文件大小须在10MiB内"); err != nil {
		return nil, err
	}
	records, err := parseImportFile(data, strings.ToLower(f.Filename))
	if err != nil {
		if ec := errcode.AsError(err); ec != nil {
			return nil, ec
		}
		return nil, errcode.New(422, "INVALID_FILE", "无法读取文件，请检查编码和文件格式")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	id := svc.S.ID()
	due, err := store.Time(e["personal_due_at"])
	if err != nil {
		return nil, fmt.Errorf("event due time: %w", err)
	}
	expiry := minTime(svc.S.Now().Add(24*time.Hour), due)
	sheetRef := "csv"
	if records[0].sheetRef != "" {
		sheetRef = records[0].sheetRef
	}
	ev := store.StrOrEmpty(e["id"])
	if err := svc.S.InsertBatch(ctx, sc, id, ev, digest, sheetRef, r["sourceLabel"], expiry); err != nil {
		return nil, err
	}
	line := 0
	for _, record := range records {
		line++
		rawJSON, err := record.json()
		if err != nil {
			return nil, err
		}
		fileErrors := []any{}
		if record.formula {
			fileErrors = append(fileErrors, store.Map(
				"field", "fields", "code", "FORMULA_NOT_ALLOWED", "message", "此行含公式，请先转换为纯值"))
		}
		rowCipher, err := svc.S.Seal(sc, store.Map("raw", store.RawJSON(rawJSON), "fileErrors", fileErrors))
		if err != nil {
			return nil, err
		}
		if err := svc.S.InsertImportRow(ctx, sc, svc.S.ID(), ev, id, strconv.Itoa(line), line,
			rowCipher, svc.S.Crypto.Hash(rawJSON)); err != nil {
			return nil, err
		}
	}
	b, err := svc.batchLocked(ctx, sc, store.Map("eventId", ev, "batchId", id))
	if err != nil {
		return nil, err
	}
	return svc.batchView(ctx, sc, b)
}

// batchView renders the batch summary; availableColumns keeps the stored
// header order of the first raw row.
func (svc *Service) batchView(ctx context.Context, sc *store.Scope, b map[string]any) (map[string]any, error) {
	rows, err := svc.S.BatchRows(ctx, sc, store.StrOrEmpty(b["id"]))
	if err != nil {
		return nil, err
	}
	columns := []any{}
	if len(rows) > 0 {
		plain, err := svc.S.UnsealPlain(rows[0]["row_cipher"])
		if err != nil {
			return nil, err
		}
		rawText, err := rawJSONFromEnvelope(plain)
		if err != nil {
			return nil, err
		}
		keys, err := rawKeys(rawText)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			columns = append(columns, store.Map("code", k, "label", k))
		}
	}
	counts := map[string]int{}
	for _, x := range rows {
		counts[store.StrOrEmpty(x["status"])]++
	}
	out := store.Map(
		"batchId", b["id"],
		"eventId", b["event_id"],
		"sourceId", b["source_id"],
		"fileDigest", b["file_digest"],
		"status", svc.value("BatchStatus", b["status"]),
		"previewVersion", b["preview_version"],
		"validRows", counts["VALID"],
		"errorRows", counts["ERROR"],
		"suspectRows", counts["DUPLICATE_SUSPECT"],
		"expiresAt", b["expires_at"],
		"version", b["version"],
		"policyVersion", b["policy_version"],
		"availableColumns", columns,
	)
	if b["mapping"] != nil {
		out["mapping"] = svc.mappingView(b["mapping"])
	}
	if b["eligibility"] != nil {
		out["eligibility"] = svc.value("SourceEligibility", b["eligibility"])
	}
	return out, nil
}

// normalizeImport maps raw columns to party fields (IntegrationService.normalize).
// Phone/email values are contact-normalized; TOTAL counts include the lead.
func normalizeImport(raw map[string]any, mapping map[string]any) (map[string]any, error) {
	fields := store.Map("companionCount", 0)
	for _, f := range []string{"name", "phone", "email"} {
		col := mapping[f+"Column"]
		if col == nil {
			continue
		}
		val := strings.TrimSpace(store.StrOrEmpty(raw[store.StrOrEmpty(col)]))
		if val == "" {
			continue
		}
		if f == "phone" {
			val = normalizeContact(val, "SMS")
		} else if f == "email" {
			val = normalizeContact(val, "EMAIL")
		}
		fields[f] = val
	}
	if col := mapping["countColumn"]; col != nil {
		val := strings.TrimSpace(store.StrOrEmpty(raw[store.StrOrEmpty(col)]))
		n := 0
		if val != "" {
			parsed, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("count parse: %w", err)
			}
			n = parsed
		}
		if store.StrOrEmpty(mapping["countMode"]) == "TOTAL" {
			if err := errcode.Require(n >= 1, 422, "INVALID_COUNT", "总人数至少为1"); err != nil {
				return nil, err
			}
			n--
		}
		if err := errcode.Require(n >= 0 && n < 1000000, 422, "INVALID_COUNT", "人数超出范围"); err != nil {
			return nil, err
		}
		fields["companionCount"] = n
	}
	return fields, nil
}

func (svc *Service) batchPreview(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := svc.intakeOpen(e); err != nil {
		return nil, err
	}
	b, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	if err := store.Version(b, r["version"]); err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(b["status"]) != "COMMITTED",
		409, "BATCH_STATE_CONFLICT", "批次已提交"); err != nil {
		return nil, err
	}
	mapping := store.Parse(r["mapping"])
	view, err := svc.batchView(ctx, sc, b)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, x := range store.List(view["availableColumns"]) {
		known[store.StrOrEmpty(store.Parse(x)["code"])] = true
	}
	for _, k := range []string{"nameColumn", "phoneColumn", "emailColumn", "countColumn"} {
		if v, has := mapping[k]; has {
			if err := errcode.Require(known[store.StrOrEmpty(v)], 422, "MAPPING_INVALID", "映射列不存在"); err != nil {
				return nil, err
			}
		}
	}
	ev := store.StrOrEmpty(e["id"])
	candidates, err := svc.S.ListParties(ctx, sc, ev)
	if err != nil {
		return nil, err
	}
	candidateFields := make([]map[string]any, 0, len(candidates))
	for _, p := range candidates {
		f, err := svc.S.Unseal(p["fields_cipher"])
		if err != nil {
			return nil, err
		}
		candidateFields = append(candidateFields, f)
	}
	scope, err := svc.sourceScope(ctx, sc, b)
	if err != nil {
		return nil, err
	}
	rows, err := svc.S.BatchRows(ctx, sc, store.StrOrEmpty(b["id"]))
	if err != nil {
		return nil, err
	}
	observed := map[string]bool{}
	for _, row := range rows {
		plain, err := svc.S.UnsealPlain(row["row_cipher"])
		if err != nil {
			return nil, err
		}
		stored := store.Parse([]byte(plain))
		raw := store.Parse(stored["raw"])
		rawText, err := rawJSONFromEnvelope(plain)
		if err != nil {
			return nil, err
		}
		fields := store.Map("companionCount", 0)
		rowErrors := []any{}
		status := "VALID"
		// Normalization/validation failures mark the row ERROR instead of
		// failing the preview; the row stays unselectable until resolved.
		if err := func() error {
			if err := errcode.Require(len(store.List(stored["fileErrors"])) == 0,
				422, "FORMULA_NOT_ALLOWED", "此行含公式"); err != nil {
				return err
			}
			normalized, err := normalizeImport(raw, mapping)
			if err != nil {
				return err
			}
			fields = normalized
			rules := store.Parse(store.Parse(e["policy"])["fields"])
			if store.Parse(rules["companionCount"])["enabled"] != true {
				if _, has := mapping["countColumn"]; !has {
					delete(fields, "companionCount")
				}
			}
			if err := svc.C.Validate("PartyInput", fields); err != nil {
				return err
			}
			return svc.validateFields(e, fields)
		}(); err != nil {
			status = "ERROR"
			rowErrors = append(rowErrors, store.Map(
				"field", "fields", "code", "INVALID_FIELD", "message", "请检查映射、必填字段和人数"))
		}
		key := fmt.Sprintf("%x", svc.S.Crypto.Hash(store.StrOrEmpty(row["source_record_id"])))
		links, err := svc.S.SourceLinks(ctx, sc, scope, key)
		if err != nil {
			return nil, err
		}
		if len(links) > 0 {
			if bytes.Equal(links[0]["source_content_digest"].([]byte), row["content_digest"].([]byte)) {
				status = "ALREADY_LINKED"
			} else {
				status = "ERROR"
				rowErrors = append(rowErrors, store.Map(
					"field", "sourceRecordId", "code", "SOURCE_CHANGED", "message", "同源记录内容已变化，请负责人核对既有组"))
			}
		}
		if status == "VALID" {
			for _, f := range []string{"phone", "email"} {
				v, has := fields[f]
				if !has || v == nil {
					continue
				}
				vs := store.StrOrEmpty(v)
				duplicateKey := f + ":" + vs
				if observed[duplicateKey] {
					status = "DUPLICATE_SUSPECT"
				}
				observed[duplicateKey] = true
				for _, other := range candidateFields {
					if other[f] != nil && vs == store.StrOrEmpty(other[f]) {
						status = "DUPLICATE_SUSPECT"
					}
				}
			}
		}
		errorsJSON, err := store.Stringify(rowErrors)
		if err != nil {
			return nil, err
		}
		rowCipher, err := svc.S.Seal(sc, store.Map(
			"raw", store.RawJSON(rawText),
			"fields", fields,
			"fileErrors", store.List(stored["fileErrors"])))
		if err != nil {
			return nil, err
		}
		if err := svc.S.UpdateImportRowPreview(ctx, sc, rowCipher, status, errorsJSON,
			1+store.Num(fields["companionCount"]), store.StrOrEmpty(row["id"])); err != nil {
			return nil, err
		}
	}
	mappingJSON, err := store.Stringify(mapping)
	if err != nil {
		return nil, err
	}
	previewDigest := svc.S.Crypto.Hash(mappingJSON + ":" + store.StrOrEmpty(e["policy_version"]))
	if err := svc.S.UpdateBatchPreview(ctx, sc, mappingJSON, store.StrOrEmpty(r["eligibility"]),
		e["policy_version"], previewDigest, store.StrOrEmpty(b["id"])); err != nil {
		return nil, err
	}
	fresh, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	return svc.batchView(ctx, sc, fresh)
}

func (svc *Service) batchRowsView(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	if _, err := svc.intEvent(ctx, sc, r, c); err != nil {
		return nil, err
	}
	b, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	rows, err := svc.S.BatchRows(ctx, sc, store.StrOrEmpty(b["id"]))
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, x := range rows {
		if _, has := r["status"]; has && store.StrOrEmpty(r["status"]) != store.StrOrEmpty(x["status"]) {
			continue
		}
		stored, err := svc.S.Unseal(x["row_cipher"])
		if err != nil {
			return nil, err
		}
		fields := store.Parse(stored["fields"])
		if stored["fields"] == nil {
			fields = store.Map("companionCount", 0)
		}
		out = append(out, store.Map(
			"rowId", x["id"],
			"sourceRecordId", x["source_record_id"],
			"lineNumber", x["line_number"],
			"fields", fields,
			"requestedSize", max(1, store.Num(x["requested_size"])),
			"status", svc.value("RowStatus", x["status"]),
			"errors", store.List(x["errors"]),
			"linkedPartyId", x["linked_party_id"],
		))
	}
	return store.Page(out, r), nil
}

func (svc *Service) duplicateResolve(ctx context.Context, sc *store.Scope, r map[string]any, c *Req) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := svc.intakeOpen(e); err != nil {
		return nil, err
	}
	b, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	if err := svc.previewVersion(b, r); err != nil {
		return nil, err
	}
	row, err := svc.S.ImportRowByIDForUpdate(ctx, sc, store.StrOrEmpty(r["rowId"]), store.StrOrEmpty(b["id"]))
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(row["status"]) == "DUPLICATE_SUSPECT",
		409, "ROW_STATE_CONFLICT", "此行不需要重复核对"); err != nil {
		return nil, err
	}
	resolution := store.StrOrEmpty(r["resolution"])
	var linked any
	if resolution == "LINK_EXISTING" {
		p, err := svc.partyRaw(ctx, sc, store.StrOrEmpty(e["id"]), store.StrOrEmpty(r["existingPartyId"]))
		if err != nil {
			return nil, err
		}
		linked = p["id"]
	}
	reasonCipher, err := svc.S.Seal(sc, store.Map("reason", r["reason"]))
	if err != nil {
		return nil, err
	}
	status := "VALID"
	if resolution == "EXCLUDE" {
		status = "EXCLUDED"
	}
	if err := svc.S.ResolveImportRow(ctx, sc, resolution, linked, reasonCipher, status, store.StrOrEmpty(row["id"])); err != nil {
		return nil, err
	}
	if err := svc.S.BumpBatchPreview(ctx, sc, store.StrOrEmpty(b["id"])); err != nil {
		return nil, err
	}
	fresh, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	return svc.batchView(ctx, sc, fresh)
}

// batchCommit submits the selected rows atomically: capacity, source-link and
// state conflicts abort the whole batch (the request transaction rolls back).
func (svc *Service) batchCommit(ctx context.Context, sc *store.Scope, r map[string]any, c *Req, entry string) (any, error) {
	e, err := svc.intEvent(ctx, sc, r, c)
	if err != nil {
		return nil, err
	}
	if err := errcode.Require(store.StrOrEmpty(e["status"]) == "READY",
		409, "EVENT_STATE_CONFLICT", "仅已发布准备中的活动可提交名单"); err != nil {
		return nil, err
	}
	b, err := svc.batchLocked(ctx, sc, r)
	if err != nil {
		return nil, err
	}
	if err := svc.previewVersion(b, r); err != nil {
		return nil, err
	}
	if err := errcode.Require(store.Num(e["policy_version"]) == store.Num(b["policy_version"]),
		409, "PREVIEW_CONFLICT", "活动字段规则已变化"); err != nil {
		return nil, err
	}
	ev := store.StrOrEmpty(e["id"])
	if b["source_id"] != nil {
		source, err := svc.S.SourceLockByID(ctx, sc, store.StrOrEmpty(b["source_id"]))
		if err != nil {
			return nil, err
		}
		if err := errcode.Require(store.Num(source["version"]) == store.Num(b["source_version"]),
			409, "PREVIEW_CONFLICT", "来源配置已变化"); err != nil {
			return nil, err
		}
		if _, err := svc.S.ConnectionAllowed(ctx, sc, store.StrOrEmpty(source["connection_id"]), ev); err != nil {
			return nil, err
		}
	}
	selected := map[string]bool{}
	for _, id := range store.List(r["selectedRowIds"]) {
		selected[store.StrOrEmpty(id)] = true
	}
	allRows, err := svc.S.BatchRows(ctx, sc, store.StrOrEmpty(b["id"]))
	if err != nil {
		return nil, err
	}
	chosen := []map[string]any{}
	for _, x := range allRows {
		if selected[store.StrOrEmpty(x["id"])] {
			chosen = append(chosen, x)
		}
	}
	if err := errcode.Require(len(chosen) == len(selected), 409, "PREVIEW_CONFLICT", "存在未知预览行"); err != nil {
		return nil, err
	}
	eligible := store.StrOrEmpty(b["eligibility"]) == "ELIGIBLE"
	required := 0
	for _, row := range chosen {
		if err := errcode.Require(store.StrOrEmpty(row["status"]) == "VALID",
			409, "ROW_STATE_CONFLICT", "选中行包含错误或未经核对的重复"); err != nil {
			return nil, err
		}
		stored, err := svc.S.Unseal(row["row_cipher"])
		if err != nil {
			return nil, err
		}
		if err := svc.validateFields(e, store.Parse(stored["fields"])); err != nil {
			return nil, err
		}
		if row["linked_party_id"] == nil && eligible {
			required += store.Num(row["requested_size"])
		}
	}
	if err := errcode.Require(store.Num(e["allocated_count"])+required <= store.Num(e["capacity"]),
		409, "CAPACITY_EXCEEDED", "剩余名额不足，整批未提交"); err != nil {
		return nil, err
	}
	// Lock linked parties in a stable order to avoid deadlocks.
	lockSet := map[string]bool{}
	for _, row := range chosen {
		if row["linked_party_id"] != nil {
			lockSet[store.StrOrEmpty(row["linked_party_id"])] = true
		}
	}
	lockIds := make([]string, 0, len(lockSet))
	for id := range lockSet {
		lockIds = append(lockIds, id)
	}
	sort.Strings(lockIds)
	for _, id := range lockIds {
		if _, err := svc.partyRaw(ctx, sc, ev, id); err != nil {
			return nil, err
		}
	}
	scope, err := svc.sourceScope(ctx, sc, b)
	if err != nil {
		return nil, err
	}
	created, linked := 0, 0
	for _, row := range chosen {
		partyID := store.StrOrEmpty(row["linked_party_id"])
		key := fmt.Sprintf("%x", svc.S.Crypto.Hash(store.StrOrEmpty(row["source_record_id"])))
		existing, err := svc.S.SourceLinkIDs(ctx, sc, scope, key)
		if err != nil {
			return nil, err
		}
		if err := errcode.Require(len(existing) == 0, 409, "SOURCE_CONFLICT", "同源记录已关联，整批未提交"); err != nil {
			return nil, err
		}
		if partyID == "" {
			stored, err := svc.S.Unseal(row["row_cipher"])
			if err != nil {
				return nil, err
			}
			fresh, err := svc.S.EventByIDForUpdate(ctx, sc, ev)
			if err != nil {
				return nil, err
			}
			sourceType := "FILE"
			if b["source_id"] != nil {
				sp, err := svc.S.SourceProvider(ctx, sc, store.StrOrEmpty(b["source_id"]))
				if err != nil {
					return nil, err
				}
				sourceType = store.StrOrEmpty(sp["provider"])
			}
			p, err := svc.partyCreate(ctx, sc, c, fresh, store.Parse(stored["fields"]), sourceType,
				eligible, "import-"+store.StrOrEmpty(b["policy_version"]), entry)
			if err != nil {
				return nil, err
			}
			partyID = store.StrOrEmpty(p["partyId"])
			created++
		} else {
			linked++
		}
		if err := svc.S.InsertSourceLink(ctx, sc, svc.S.ID(), ev, partyID, scope, key,
			row["content_digest"].([]byte), store.StrOrEmpty(b["id"]), svc.S.Now()); err != nil {
			return nil, err
		}
		if err := svc.S.CommitImportRow(ctx, sc, partyID, store.StrOrEmpty(row["id"])); err != nil {
			return nil, err
		}
	}
	if err := svc.S.CommitBatch(ctx, sc, entry, store.StrOrEmpty(b["id"])); err != nil {
		return nil, err
	}
	return store.Map(
		"batchId", b["id"],
		"operationId", r["operationId"],
		"createdCount", created,
		"linkedCount", linked,
		"allocatedPeople", required,
		"replayed", false,
	), nil
}
